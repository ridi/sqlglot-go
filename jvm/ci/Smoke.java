// Single-file FFM smoke test for a built libsqlglot.{so,dylib}, used by CI to verify a
// cross-compiled native library actually loads and runs on the target platform.
// Usage: java --enable-native-access=ALL-UNNAMED jvm/ci/Smoke.java <path-to-libsqlglot>
import java.lang.foreign.Arena;
import java.lang.foreign.FunctionDescriptor;
import java.lang.foreign.Linker;
import java.lang.foreign.MemorySegment;
import java.lang.foreign.SymbolLookup;
import java.lang.foreign.ValueLayout;
import java.lang.invoke.MethodHandle;

public class Smoke {
    public static void main(String[] args) throws Throwable {
        Linker linker = Linker.nativeLinker();
        SymbolLookup lookup = SymbolLookup.libraryLookup(args[0], Arena.global());
        MethodHandle probe = linker.downcallHandle(
            lookup.find("ProbeJSON").orElseThrow(),
            FunctionDescriptor.of(ValueLayout.ADDRESS, ValueLayout.ADDRESS, ValueLayout.ADDRESS, ValueLayout.ADDRESS));
        MethodHandle free = linker.downcallHandle(
            lookup.find("FreeCString").orElseThrow(),
            FunctionDescriptor.ofVoid(ValueLayout.ADDRESS));
        String schema = "{\"users\":{\"id\":\"BIGINT\",\"rrn\":\"VARCHAR\"}}";
        try (Arena a = Arena.ofConfined()) {
            MemorySegment res = (MemorySegment) probe.invoke(
                a.allocateFrom("SELECT id, rrn FROM users WHERE rrn = 'x'"),
                a.allocateFrom("postgres"),
                a.allocateFrom(schema));
            String json = res.reinterpret(Long.MAX_VALUE).getString(0);
            free.invoke(res);
            System.out.println("RESULT: " + json);
            if (!json.contains("\"resolved\":true") || !json.contains("users.rrn")) {
                System.err.println("SMOKE FAIL");
                System.exit(1);
            }
            System.out.println("SMOKE OK: " + args[0]);
        }
    }
}
