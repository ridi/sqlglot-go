package dialects_test

import (
	"strings"
	"testing"

	"github.com/sjincho/sqlglot-go/dialects"
)

func TestMySQLVersionSetting(t *testing.T) {
	tests := []struct {
		name    string
		dialect string
		want    int
	}{
		{name: "major minor patch", dialect: "mysql, mysql_version=8.0.33", want: 80033},
		{name: "major minor", dialect: "mysql, mysql_version=8.4", want: 80400},
		{name: "mysql 5 major minor patch", dialect: "mysql, mysql_version=5.7.33", want: 50733},
		{name: "mysql 5 major minor", dialect: "mysql, mysql_version=5.7", want: 50700},
		{name: "major only", dialect: "mysql, mysql_version=8", want: 80000},
		{name: "patch suffix", dialect: "mysql, mysql_version=8.4.9-1.el9", want: 80409},
		{name: "major suffix", dialect: "mysql, mysql_version=8-log", want: 80000},
		{name: "minor suffix", dialect: "mysql, mysql_version=8.4-log", want: 80400},
		{name: "whitespace", dialect: " mysql , mysql_version = 8.0.33 ", want: 80033},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := dialects.GetOrRaise(tt.dialect)
			if err != nil {
				t.Fatalf("GetOrRaise(%q): %v", tt.dialect, err)
			}
			if d.MySQLVersion == nil {
				t.Fatalf("MySQLVersion = nil, want %d", tt.want)
			}
			if got := *d.MySQLVersion; got != tt.want {
				t.Fatalf("MySQLVersion = %d, want %d", got, tt.want)
			}
			if !d.TokenizerConfig.MySQLExecutableComments {
				t.Fatal("MySQLExecutableComments = false, want true")
			}
		})
	}

	bare, err := dialects.GetOrRaise("mysql")
	if err != nil {
		t.Fatalf("GetOrRaise(mysql): %v", err)
	}
	if bare.MySQLVersion != nil {
		t.Fatalf("bare MySQLVersion = %d, want nil", *bare.MySQLVersion)
	}
}

func TestMySQLVersionSettingErrors(t *testing.T) {
	tests := []struct {
		name      string
		dialect   string
		wantError string
	}{
		{name: "missing value delimiter", dialect: "mysql, mysql_version", wantError: `dialect setting "mysql_version" requires a value`},
		{name: "empty", dialect: "mysql, mysql_version=", wantError: "mysql_version"},
		{name: "non numeric start", dialect: "mysql, mysql_version=release-8.0.33", wantError: "mysql_version"},
		{name: "missing minor", dialect: "mysql, mysql_version=8.", wantError: "mysql_version"},
		{name: "missing patch", dialect: "mysql, mysql_version=8.4.", wantError: "mysql_version"},
		{name: "doubled dot", dialect: "mysql, mysql_version=8..4", wantError: "mysql_version"},
		{name: "oversized minor", dialect: "mysql, mysql_version=8.100", wantError: "mysql_version"},
		{name: "oversized patch", dialect: "mysql, mysql_version=8.4.100", wantError: "mysql_version"},
		{name: "overflow", dialect: "mysql, mysql_version=" + strings.Repeat("9", 100), wantError: "mysql_version"},
		{name: "unknown setting", dialect: "mysql, future_setting=1", wantError: `unsupported dialect setting "future_setting" (supported: normalization_strategy, mysql_version)`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dialects.GetOrRaise(tt.dialect)
			if err == nil {
				t.Fatalf("GetOrRaise(%q) succeeded, want error", tt.dialect)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("GetOrRaise(%q) error = %q, want substring %q", tt.dialect, err, tt.wantError)
			}
		})
	}
}

func TestMySQLVersionSettingAcceptedButInertForOtherDialects(t *testing.T) {
	for _, dialect := range []string{
		", mysql_version=8.0.33",
		"postgres, mysql_version=8.0.33",
	} {
		d, err := dialects.GetOrRaise(dialect)
		if err != nil {
			t.Fatalf("GetOrRaise(%q): %v", dialect, err)
		}
		if d.MySQLVersion == nil || *d.MySQLVersion != 80033 {
			t.Fatalf("GetOrRaise(%q) MySQLVersion = %v, want 80033", dialect, d.MySQLVersion)
		}
		if d.TokenizerConfig.MySQLExecutableComments {
			t.Fatalf("GetOrRaise(%q) MySQLExecutableComments = true, want false", dialect)
		}
	}

	for _, dialect := range []string{
		", mysql_version=8.",
		"postgres, mysql_version=8.",
	} {
		if _, err := dialects.GetOrRaise(dialect); err == nil {
			t.Fatalf("GetOrRaise(%q) succeeded, want malformed mysql_version error", dialect)
		} else if !strings.Contains(err.Error(), "mysql_version") {
			t.Fatalf("GetOrRaise(%q) error = %q, want mysql_version", dialect, err)
		}
	}
}
