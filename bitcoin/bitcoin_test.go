package bitcoin

import (
	"sync"
	"testing"
)

func TestRandString(t *testing.T) {
	testData := []struct {
		length int
	}{
		{5},
		{10},
	}
	for i, td := range testData {
		str := randString(td.length)
		if len(str) != td.length {
			t.Errorf("%d. randString(%d) len = %d", i, td.length, len(str))
		}
	}
}

func TestNewClient(t *testing.T) {
	launchOnce = sync.Once{}
	launchWG = sync.WaitGroup{}
	launchWG.Add(1)
	cl, err := NewClient()
	if err != nil {
		Kill()
		t.Fatal(err)
	}
	cl2, err := NewClient()
	if err != nil {
		Kill()
		t.Fatal(err)
	}
	Kill()
	if cl != cl2 {
		t.Fatalf("NewClient() not returning same client each time %#v %#v", cl, cl2)
	}
}
