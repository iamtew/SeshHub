package config

import (
	"flag"
	"testing"
)

func TestListenAddr(t *testing.T) {
	if (Config{Port: "53054"}).ListenAddr() != ":53054" {
		t.Fatal("port")
	}
	if (Config{Port: "127.0.0.1:53054"}).ListenAddr() != "127.0.0.1:53054" {
		t.Fatal("addr")
	}
}

func TestApplyFlagsOverridesPort(t *testing.T) {
	c := Config{Port: "53053", DatabaseURL: "file:seshhub.db"}
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	if err := c.ApplyFlags(fs, []string{"-port", "127.0.0.1:53054", "-db", ":memory:"}); err != nil {
		t.Fatal(err)
	}
	if c.Port != "127.0.0.1:53054" || c.ListenAddr() != "127.0.0.1:53054" || c.DatabaseURL != ":memory:" {
		t.Fatalf("%+v", c)
	}
}
