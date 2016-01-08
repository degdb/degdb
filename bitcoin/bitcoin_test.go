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
		t.Error(err)
	}
	cl2, err := NewClient()
	if err != nil {
		t.Error(err)
	}
	if cl != cl2 {
		t.Errorf("NewClient() not returning same client each time %#v %#v", cl, cl2)
	}
	Kill()
}
