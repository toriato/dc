package dc

import (
	"strconv"
	"strings"
)

const decodeKey = "yL/M=zNa0bcPQdReSfTgUhViWjXkYIZmnpo+qArOBslCt2D3uE4Fv5G6wH178xJ9K"

func decode(keys, code string) string {
	// common.js?v=210817:858
	k := [4]byte{}
	o := strings.Builder{}

	for c := 0; c < len(keys); {
		for i := 0; i < len(k); i++ {
			k[i] = byte(strings.Index(decodeKey, string(keys[c])))
			c += 1
		}

		o.WriteByte(k[0]<<2 | k[1]>>4)

		if k[2] != 64 {
			o.WriteByte((15&k[1])<<4 | k[2]>>2)
		}

		if k[3] != 64 {
			o.WriteByte((3&k[2])<<6 | k[3])
		}
	}

	// common.js?v=210817:862
	keys = o.String()
	fi, _ := strconv.Atoi(keys[0:1])

	if fi > 5 {
		fi -= 5
	} else {
		fi += 4
	}

	keys = string(rune(fi)) + keys[1:]

	// common.js?v=210817:859
	o.Reset()
	o.WriteString(code[:len(code)-10])

	for idx, s := range strings.Split(keys, ",") {
		key, _ := strconv.ParseFloat(s, 64)
		o.WriteByte(byte(2 * (key - float64(idx) - 1) / float64(13-idx-1)))
	}

	return o.String()
}
