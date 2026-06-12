package apply

import (
	"testing"

	"github.com/spf13/viper"
)

func TestSubtractMACs(t *testing.T) {
	got := subtractMACs([]string{"a", "b", "c", "d"}, []string{"b", "d"})
	want := []string{"a", "c"}
	if len(got) != len(want) || got[0] != "a" || got[1] != "c" {
		t.Errorf("subtractMACs = %v, want %v", got, want)
	}
	if all := subtractMACs([]string{"a"}, nil); len(all) != 1 || all[0] != "a" {
		t.Errorf("subtractMACs with empty remove = %v, want [a]", all)
	}
}

func TestResolveApplyVerifyDefaultsTrue(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	if !resolveApplyVerify("mist") {
		t.Error("apply_verify should default to true when unset")
	}
	viper.Set("api.meraki-big.apply_verify", false)
	if resolveApplyVerify("meraki-big") {
		t.Error("apply_verify should honor a per-API false override")
	}
	if !resolveApplyVerify("mist") {
		t.Error("other APIs should still default to true")
	}
}
