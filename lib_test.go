package gameserver  
import (
	"testing"
)

func TestCheckBody(t *testing.T) {
	msg := Msg{Body: map[string]interface{}{
			"int": 234,
			"badint": "bad",
			"string": "yo",
			"badstring": 23,
		}}
	
	var f float64 = 2
	msg.Body["f64"] = f

	var myint int	
	var mystr string

	type Case struct {
		destInt *int
		destStr *string
		key string
	}
	tests := map[Case]bool{
		{destInt: &myint, key: "int"}: true,
		{destStr: &mystr, key: "string"}: true,
		{destInt: &myint, key: "badint"}: false,
		{destStr: &mystr, key: "badstring"}: false,
		{destStr: &mystr, key: "nostring"}: false,
		{destInt: &myint, key: "f64"}: true,
	}

	for c, success := range tests {
		var err error 
		if c.destInt != nil {
			err = CheckNumber(msg, c.key, c.destInt)
		} else if c.destStr != nil {
			err = CheckBody(msg, c.key, c.destStr)
		}
		if err == nil && !success || 
			err != nil && success {
			t.Errorf("Expected %v got %v for case %v", success, err , c)
		}
		if err == nil && c.destInt != nil && *c.destInt == 0 {
			t.Errorf("Expected destination to change, but got 0")
		}
	}
}
