package cleancode

// calc works, and it is exactly the kind of code you are afraid to touch.
func calc(d []map[string]int, c string, f bool) (int, int, int, int) {
	t := 0
	for _, x := range d {
		t += x["p"] * x["q"]
	}
	r := 0
	if c == "WELCOME10" && f {
		r = t * 10 / 100
	} else if c == "BIG20" && t >= 50000 {
		r = t * 20 / 100
	}
	s := 2000
	if t-r >= 20000 {
		s = 0
	}
	return t, r, s, t - r + s
}
