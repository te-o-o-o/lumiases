package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
)

// frameHead is a bust rendered by tools/turntable.py: one picture per yaw
// step, nod angle and expression. A pose is read by blending the frames
// around it, so turning shows the real profile, no mirror needed.
type frameHead struct {
	step    float64 // degrees between yaw frames
	pitches []int   // nod angles, ascending
	// exprs maps an expression ("neutral", "frown"...) to its frames by
	// yaw index, then pitch index.
	exprs map[string][][]*faceImg
	w, h  int
}

// loadFrames reads every <expression>-y<yaw>-p<pitch>.png of dir. Every
// expression must cover the same yaws and pitches as "neutral".
func loadFrames(dir string) (*frameHead, error) {
	paths, _ := filepath.Glob(filepath.Join(dir, "*-y*-p*.png"))
	if len(paths) == 0 {
		return nil, fmt.Errorf("%s: no frames (expected names like neutral-y030-p0.png, from tools/turntable.py)", dir)
	}
	type key struct {
		expr       string
		yaw, pitch int
	}
	keys := make([]key, len(paths))
	yaws, pitches := map[int]bool{}, map[int]bool{}
	for i, p := range paths {
		name := strings.TrimSuffix(filepath.Base(p), ".png")
		cut := strings.LastIndex(name, "-y")
		k := key{expr: name[:cut]}
		if _, err := fmt.Sscanf(name[cut:], "-y%d-p%d", &k.yaw, &k.pitch); err != nil {
			return nil, fmt.Errorf("%s: unexpected frame name", p)
		}
		keys[i], yaws[k.yaw], pitches[k.pitch] = k, true, true
	}

	// The front frame sets the exposure for all; then decode the rest in
	// parallel, as a head is often a hundred PNGs.
	var tone [3]float64
	front := slices.IndexFunc(keys, func(k key) bool { return k.expr == "neutral" && k.yaw == 0 && k.pitch == 0 })
	if front < 0 {
		return nil, fmt.Errorf("%s: no neutral-y000-p0.png to set the exposure", dir)
	}
	if _, err := loadFace(paths[front], [4]float64{0, 0, 1, 1}, &tone); err != nil {
		return nil, err
	}
	imgs := make([]*faceImg, len(paths))
	errs := make([]error, len(paths))
	var wg sync.WaitGroup
	for i, p := range paths {
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			imgs[i], errs[i] = loadFace(p, [4]float64{0, 0, 1, 1}, &tone)
		}(i, p)
	}
	wg.Wait()

	f := &frameHead{exprs: map[string][][]*faceImg{}}
	yawList := sortedKeys(yaws)
	f.pitches = sortedKeys(pitches)
	if len(yawList) > 1 {
		f.step = float64(yawList[1] - yawList[0])
	} else {
		f.step = 360
	}
	for i, k := range keys {
		if errs[i] != nil {
			return nil, errs[i]
		}
		set := f.exprs[k.expr]
		if set == nil {
			set = make([][]*faceImg, len(yawList))
			for y := range set {
				set[y] = make([]*faceImg, len(f.pitches))
			}
			f.exprs[k.expr] = set
		}
		set[sort.SearchInts(yawList, k.yaw)][sort.SearchInts(f.pitches, k.pitch)] = imgs[i]
		f.w, f.h = imgs[i].w, imgs[i].h
	}
	for expr, set := range f.exprs {
		for _, row := range set {
			for _, img := range row {
				if img == nil {
					return nil, fmt.Errorf("%s: expression %q misses frames", dir, expr)
				}
				if img.w != f.w || img.h != f.h {
					return nil, fmt.Errorf("%s: frames of different sizes", dir)
				}
			}
		}
	}
	if f.exprs["neutral"] == nil {
		return nil, fmt.Errorf("%s: no neutral frames", dir)
	}
	return f, nil
}

func sortedKeys(m map[int]bool) []int {
	s := make([]int, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	sort.Ints(s)
	return s
}

// at reads luminance and mask at u, v for a pose: yaw in radians, pitch in
// degrees, and how much of the frown and smile expressions to blend in.
func (f *frameHead) at(u, v, yaw, pitch, frown, smile float64) (float64, float64) {
	n := len(f.exprs["neutral"])
	deg := math.Mod(yaw*180/math.Pi, 360)
	if deg < 0 {
		deg += 360
	}
	fy := deg / f.step
	y0 := int(fy) % n
	y1, wy := (y0+1)%n, fy-math.Floor(fy)
	// Between the two nearest nod angles, clamped to the rendered ones.
	p0, p1, wp := 0, 0, 0.0
	if last := len(f.pitches) - 1; pitch <= float64(f.pitches[0]) {
		p0, p1 = 0, 0
	} else if pitch >= float64(f.pitches[last]) {
		p0, p1 = last, last
	} else {
		p1 = sort.SearchInts(f.pitches, int(math.Ceil(pitch)))
		p0 = p1 - 1
		wp = (pitch - float64(f.pitches[p0])) / float64(f.pitches[p1]-f.pitches[p0])
	}
	read := func(set [][]*faceImg) (float64, float64) {
		l00, m00 := set[y0][p0].at(u, v)
		l10, m10 := set[y1][p0].at(u, v)
		l01, m01 := set[y0][p1].at(u, v)
		l11, m11 := set[y1][p1].at(u, v)
		return lerp(lerp(l00, l10, wy), lerp(l01, l11, wy), wp), lerp(lerp(m00, m10, wy), lerp(m01, m11, wy), wp)
	}
	l, m := read(f.exprs["neutral"])
	if set := f.exprs["frown"]; set != nil && frown > 0.01 {
		l2, m2 := read(set)
		l, m = lerp(l, l2, frown), lerp(m, m2, frown)
	}
	if set := f.exprs["smile"]; set != nil && smile > 0.01 {
		l2, m2 := read(set)
		l, m = lerp(l, l2, smile), lerp(m, m2, smile)
	}
	return l, m
}

// loadHead reads a statue: a folder of rendered frames, or one photo.
func loadHead(path string, crop [4]float64) (*head, error) {
	if st, err := os.Stat(path); err == nil && st.IsDir() {
		f, err := loadFrames(path)
		return &head{frames: f}, err
	}
	pic, err := loadFace(path, crop, nil)
	return &head{photo: pic}, err
}
