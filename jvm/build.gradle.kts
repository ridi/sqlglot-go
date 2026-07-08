import org.gradle.api.tasks.Exec
import org.gradle.language.jvm.tasks.ProcessResources

plugins {
    kotlin("jvm") version "2.2.0"
    `java-library`
    `maven-publish`
}

// The JVM binding for sqlglot-go. Group/version are placeholders — a consumer using git subtree +
// includeBuild() ignores them; set real coordinates before publishing to a Maven repository.
group = "io.github.sjincho"
version = "0.1.0-SNAPSHOT"

repositories { mavenCentral() }

// FFM (java.lang.foreign) is stable since JDK 22; proxy-monster runs JDK 24.
kotlin { jvmToolchain(24) }

dependencies {
    testImplementation(kotlin("test"))
    testImplementation("org.junit.jupiter:junit-jupiter:5.10.2")
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
}

// ---- Native libraries: build cmd/libsqlglot (Go, c-shared) per target and bundle as resources ----
// The Go module root is the parent of this Gradle project (jvm/ is its own build).
//
// One build produces libs for all supported targets so a single (fat) jar runs on any of them:
// the host platform is built natively; Linux targets are cross-compiled with Zig as the C compiler
// (`zig cc -target …`), pinned to an old glibc for forward compatibility. The FFM wrapper selects
// `native/<os>-<arch>/libsqlglot.<ext>` at runtime.
//
//   ./gradlew build                            -> host lib only (fast; dev / composite build)
//   ./gradlew build -Psqlglot.native.all=true  -> all targets (needs `zig` on PATH); the distributable jar
//
// Cross-building the darwin target from a non-darwin host is NOT supported (needs the macOS SDK),
// so the all-targets build must run on macOS/arm64. CI does exactly that.
private val goModuleDir = rootDir.parentFile
private val nativeResourcesDir = layout.buildDirectory.dir("native-resources")
private val zigExe = (findProperty("zig") as String?) ?: "zig"

// (os, arch, ext, zigTarget). zigTarget == null => build natively (host only, no cross toolchain).
data class NativeTarget(val os: String, val arch: String, val ext: String, val zigTarget: String?)

val nativeTargets = listOf(
    NativeTarget("darwin", "arm64", "dylib", null),
    NativeTarget("linux", "amd64", "so", "x86_64-linux-gnu.2.17"),
    NativeTarget("linux", "arm64", "so", "aarch64-linux-gnu.2.17"),
)

fun hostOsArch(): Pair<String, String> {
    val osName = System.getProperty("os.name").lowercase()
    val os = when {
        osName.contains("mac") || osName.contains("darwin") -> "darwin"
        osName.contains("linux") -> "linux"
        else -> error("unsupported build OS: $osName")
    }
    val arch = when (val a = System.getProperty("os.arch").lowercase()) {
        "aarch64", "arm64" -> "arm64"
        "x86_64", "amd64" -> "amd64"
        else -> error("unsupported build arch: $a")
    }
    return os to arch
}

val nativeTasks = nativeTargets.associateWith { t ->
    tasks.register<Exec>("buildNativeLib_${t.os}_${t.arch}") {
        group = "build"
        description = "Builds cmd/libsqlglot (c-shared) for ${t.os}/${t.arch}."
        val outFile = nativeResourcesDir.get().dir("native/${t.os}-${t.arch}").file("libsqlglot.${t.ext}").asFile
        workingDir = goModuleDir
        environment("CGO_ENABLED", "1")
        environment("GOOS", t.os)
        environment("GOARCH", t.arch)
        if (t.zigTarget != null) {
            // Zig as the cross C compiler for cgo. Requires `zig` on PATH (or -Pzig=/path/to/zig).
            environment("CC", "$zigExe cc -target ${t.zigTarget}")
            environment("CXX", "$zigExe c++ -target ${t.zigTarget}")
        }
        commandLine("go", "build", "-buildmode=c-shared", "-o", outFile.absolutePath, "./cmd/libsqlglot")
        doFirst { outFile.parentFile.mkdirs() }
        inputs.files(
            fileTree(goModuleDir) {
                include("**/*.go", "go.mod", "go.sum")
                exclude("jvm/**", ".reference/**", ".git/**")
            },
        ).withPathSensitivity(PathSensitivity.RELATIVE)
        if (t.zigTarget != null) inputs.property("zig", zigExe)
        outputs.file(outFile)
    }
}

val (hostOs, hostArch) = hostOsArch()
val hostTarget = nativeTargets.firstOrNull { it.os == hostOs && it.arch == hostArch }
    ?: error("no native target defined for host $hostOs/$hostArch")

val buildNativeLib by tasks.registering {
    group = "build"
    description = "Builds the c-shared lib for the host platform only (fast; used by the default build)."
    dependsOn(nativeTasks[hostTarget])
}

val buildAllNativeLibs by tasks.registering {
    group = "build"
    description = "Builds c-shared libs for ALL targets (host native + Linux via Zig). Run on macOS/arm64."
    dependsOn(nativeTasks.values)
}

private val bundleAllNatives = (findProperty("sqlglot.native.all") as String?)?.toBoolean() ?: false

sourceSets.named("main") { resources.srcDir(nativeResourcesDir) }
tasks.named<ProcessResources>("processResources") {
    dependsOn(if (bundleAllNatives) buildAllNativeLibs else buildNativeLib)
    exclude("**/*.h") // the cgo header is emitted next to the lib; not needed at runtime
}

tasks.test {
    useJUnitPlatform()
    // FFM restricted methods (libraryLookup / reinterpret) — grant native access.
    jvmArgs("--enable-native-access=ALL-UNNAMED")
}

publishing {
    publications { create<MavenPublication>("maven") { from(components["java"]) } }
}
