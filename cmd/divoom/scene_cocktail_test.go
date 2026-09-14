package main

import (
	"os"
	"testing"
)

// TestCocktailBgPathForTracksCachedPool pins the fix for the
// "only ever Mojito" bug: the container starts with an empty cache,
// so the pool is 0 and every activation lands on index 000 (drink
// 11000, Mojito). Once a push fills the cache, the running scene
// must walk every cached drink without a daemon restart.
func TestCocktailBgPathForTracksCachedPool(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	s := cocktailScene(nil)

	if got, want := s.BgPathFor(""), bgCocktailFor(0); got != want {
		t.Fatalf("empty cache: BgPathFor = %q, want %q", got, want)
	}

	dir := cocktailCacheDir()
	for _, name := range []string{"11000.json", "11005.json", "11007.json", "cat_Cocktail.json"} {
		if err := os.WriteFile(dir+"/"+name, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	seen := map[string]bool{}
	for range 3 {
		seen[s.BgPathFor("")] = true
	}
	for i := range 3 {
		if !seen[bgCocktailFor(i)] {
			t.Errorf("3 activations over a 3-drink pool never showed %q; saw %v", bgCocktailFor(i), seen)
		}
	}
}
