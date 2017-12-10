// +build gofuzz

package css

func Fuzz(data []byte) int {
	if _, err := Compile(string(data)); err != nil {
		return 0
	}
	return 1
}
