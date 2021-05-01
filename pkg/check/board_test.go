package check

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
)

func TestBoard(t *testing.T) {
	b := NewBoard()
	x, _ := json.Marshal(b.position_layout)
	ioutil.WriteFile("layout.json", x, 0600)

	fmt.Printf("%#v\n", b.position_layout)
	t.Error()
}
