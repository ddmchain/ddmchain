
package language

import "errors"

type Matcher interface {
	Match(t ...Tag) (tag Tag, index int, c Confidence)
}

func Comprehends(speaker, alternative Tag) Confidence {
	_, _, c := NewMatcher([]Tag{alternative}).Match(speaker)
	return c
}

func NewMatcher(t []Tag) Matcher {
	return newMatcher(t)
}

func (m *matcher) Match(want ...Tag) (t Tag, index int, c Confidence) {
	match, w, c := m.getBest(want...)
	if match == nil {
		t = m.default_.tag
	} else {
		t, index = match.tag, match.index
	}

	if u, ok := w.Extension('u'); ok {
		t, _ = Raw.Compose(t, u)
	}
	return t, index, c
}

type scriptRegionFlags uint8

const (
	isList = 1 << iota
	scriptInFrom
	regionInFrom
)

func (t *Tag) setUndefinedLang(id langID) {
	if t.lang == 0 {
		t.lang = id
	}
}

func (t *Tag) setUndefinedScript(id scriptID) {
	if t.script == 0 {
		t.script = id
	}
}

func (t *Tag) setUndefinedRegion(id regionID) {
	if t.region == 0 || t.region.contains(id) {
		t.region = id
	}
}

var ErrMissingLikelyTagsData = errors.New("missing likely tags data")

func (t Tag) addLikelySubtags() (Tag, error) {
	id, err := addTags(t)
	if err != nil {
		return t, err
	} else if id.equalTags(t) {
		return t, nil
	}
	id.remakeString()
	return id, nil
}

func specializeRegion(t *Tag) bool {
	if i := regionInclusion[t.region]; i < nRegionGroups {
		x := likelyRegionGroup[i]
		if langID(x.lang) == t.lang && scriptID(x.script) == t.script {
			t.region = regionID(x.region)
		}
		return true
	}
	return false
}

func addTags(t Tag) (Tag, error) {

	if t.private() {
		return t, nil
	}
	if t.script != 0 && t.region != 0 {
		if t.lang != 0 {

			specializeRegion(&t)
			return t, nil
		}

		list := likelyRegion[t.region : t.region+1]
		if x := list[0]; x.flags&isList != 0 {
			list = likelyRegionList[x.lang : x.lang+uint16(x.script)]
		}
		for _, x := range list {

			if scriptID(x.script) == t.script {
				t.setUndefinedLang(langID(x.lang))
				return t, nil
			}
		}
	}
	if t.lang != 0 {

		if t.lang < langNoIndexOffset {
			x := likelyLang[t.lang]
			if x.flags&isList != 0 {
				list := likelyLangList[x.region : x.region+uint16(x.script)]
				if t.script != 0 {
					for _, x := range list {
						if scriptID(x.script) == t.script && x.flags&scriptInFrom != 0 {
							t.setUndefinedRegion(regionID(x.region))
							return t, nil
						}
					}
				} else if t.region != 0 {
					count := 0
					goodScript := true
					tt := t
					for _, x := range list {

						if x.flags&scriptInFrom == 0 && t.region.contains(regionID(x.region)) {
							tt.region = regionID(x.region)
							tt.setUndefinedScript(scriptID(x.script))
							goodScript = goodScript && tt.script == scriptID(x.script)
							count++
						}
					}
					if count == 1 {
						return tt, nil
					}

					if goodScript {
						t.script = tt.script
					}
				}
			}
		}
	} else {

		if t.script != 0 {
			x := likelyScript[t.script]
			if x.region != 0 {
				t.setUndefinedRegion(regionID(x.region))
				t.setUndefinedLang(langID(x.lang))
				return t, nil
			}
		}

		if t.region != 0 {
			if i := regionInclusion[t.region]; i < nRegionGroups {
				x := likelyRegionGroup[i]
				if x.region != 0 {
					t.setUndefinedLang(langID(x.lang))
					t.setUndefinedScript(scriptID(x.script))
					t.region = regionID(x.region)
				}
			} else {
				x := likelyRegion[t.region]
				if x.flags&isList != 0 {
					x = likelyRegionList[x.lang]
				}
				if x.script != 0 && x.flags != scriptInFrom {
					t.setUndefinedLang(langID(x.lang))
					t.setUndefinedScript(scriptID(x.script))
					return t, nil
				}
			}
		}
	}

	if t.lang < langNoIndexOffset {
		x := likelyLang[t.lang]
		if x.flags&isList != 0 {
			x = likelyLangList[x.region]
		}
		if x.region != 0 {
			t.setUndefinedScript(scriptID(x.script))
			t.setUndefinedRegion(regionID(x.region))
		}
		specializeRegion(&t)
		if t.lang == 0 {
			t.lang = _en 
		}
		return t, nil
	}
	return t, ErrMissingLikelyTagsData
}

func (t *Tag) setTagsFrom(id Tag) {
	t.lang = id.lang
	t.script = id.script
	t.region = id.region
}

func (t Tag) minimize() (Tag, error) {
	t, err := minimizeTags(t)
	if err != nil {
		return t, err
	}
	t.remakeString()
	return t, nil
}

func minimizeTags(t Tag) (Tag, error) {
	if t.equalTags(und) {
		return t, nil
	}
	max, err := addTags(t)
	if err != nil {
		return t, err
	}
	for _, id := range [...]Tag{
		{lang: t.lang},
		{lang: t.lang, region: t.region},
		{lang: t.lang, script: t.script},
	} {
		if x, err := addTags(id); err == nil && max.equalTags(x) {
			t.setTagsFrom(id)
			break
		}
	}
	return t, nil
}

type matcher struct {
	default_     *haveTag
	index        map[langID]*matchHeader
	passSettings bool
}

type matchHeader struct {
	exact []*haveTag
	max   []*haveTag
}

type haveTag struct {
	tag Tag

	index int

	conf Confidence

	maxRegion regionID
	maxScript scriptID

	altScript scriptID

	nextMax uint16
}

func makeHaveTag(tag Tag, index int) (haveTag, langID) {
	max := tag
	if tag.lang != 0 {
		max, _ = max.canonicalize(All)
		max, _ = addTags(max)
		max.remakeString()
	}
	return haveTag{tag, index, Exact, max.region, max.script, altScript(max.lang, max.script), 0}, max.lang
}

func altScript(l langID, s scriptID) scriptID {
	for _, alt := range matchScript {
		if (alt.lang == 0 || langID(alt.lang) == l) && scriptID(alt.have) == s {
			return scriptID(alt.want)
		}
	}
	return 0
}

func (h *matchHeader) addIfNew(n haveTag, exact bool) {

	for _, v := range h.exact {
		if v.tag.equalsRest(n.tag) {
			return
		}
	}
	if exact {
		h.exact = append(h.exact, &n)
	}

	for i, v := range h.max {
		if v.maxScript == n.maxScript &&
			v.maxRegion == n.maxRegion &&
			v.tag.variantOrPrivateTagStr() == n.tag.variantOrPrivateTagStr() {
			for h.max[i].nextMax != 0 {
				i = int(h.max[i].nextMax)
			}
			h.max[i].nextMax = uint16(len(h.max))
			break
		}
	}
	h.max = append(h.max, &n)
}

func (m *matcher) header(l langID) *matchHeader {
	if h := m.index[l]; h != nil {
		return h
	}
	h := &matchHeader{}
	m.index[l] = h
	return h
}

func newMatcher(supported []Tag) *matcher {
	m := &matcher{
		index: make(map[langID]*matchHeader),
	}
	if len(supported) == 0 {
		m.default_ = &haveTag{}
		return m
	}

	for i, tag := range supported {
		pair, _ := makeHaveTag(tag, i)
		m.header(tag.lang).addIfNew(pair, true)
	}
	m.default_ = m.header(supported[0].lang).exact[0]
	for i, tag := range supported {
		pair, max := makeHaveTag(tag, i)
		if max != tag.lang {
			m.header(max).addIfNew(pair, false)
		}
	}

	update := func(want, have uint16, conf Confidence, force bool) {
		if hh := m.index[langID(have)]; hh != nil {
			if !force && len(hh.exact) == 0 {
				return
			}
			hw := m.header(langID(want))
			for _, ht := range hh.max {
				v := *ht
				if conf < v.conf {
					v.conf = conf
				}
				v.nextMax = 0 
				if v.altScript != 0 {
					v.altScript = altScript(langID(want), v.maxScript)
				}
				hw.addIfNew(v, conf == Exact && len(hh.exact) > 0)
			}
		}
	}

	for _, ml := range matchLang {
		update(ml.want, ml.have, Confidence(ml.conf), false)
		if !ml.oneway {
			update(ml.have, ml.want, Confidence(ml.conf), false)
		}
	}

	for i, lm := range langAliasMap {
		if lm.from == _sh {
			continue
		}

		conf := Exact
		if langAliasTypes[i] != langMacro {
			if !isExactEquivalent(langID(lm.from)) {
				conf = High
			}
			update(lm.to, lm.from, conf, true)
		}
		update(lm.from, lm.to, conf, true)
	}
	return m
}

func (m *matcher) getBest(want ...Tag) (got *haveTag, orig Tag, c Confidence) {
	best := bestMatch{}
	for _, w := range want {
		var max Tag

		h := m.index[w.lang]
		if w.lang != 0 {

			if h == nil {
				continue
			}
			for i := range h.exact {
				have := h.exact[i]
				if have.tag.equalsRest(w) {
					return have, w, Exact
				}
			}
			max, _ = w.canonicalize(Legacy | Deprecated)
			max, _ = addTags(max)
		} else {

			if h != nil {
				for i := range h.exact {
					have := h.exact[i]
					if have.tag.equalsRest(w) {
						return have, w, Exact
					}
				}
			}
			if w.script == 0 && w.region == 0 {

				continue
			}
			max, _ = addTags(w)
			if h = m.index[max.lang]; h == nil {
				continue
			}
		}

		for i := range h.max {
			have := h.max[i]
			best.update(have, w, max.script, max.region)
			if best.conf == Exact {
				for have.nextMax != 0 {
					have = h.max[have.nextMax]
					best.update(have, w, max.script, max.region)
				}
				return best.have, best.want, High
			}
		}
	}
	if best.conf <= No {
		if len(want) != 0 {
			return nil, want[0], No
		}
		return nil, Tag{}, No
	}
	return best.have, best.want, best.conf
}

type bestMatch struct {
	have *haveTag
	want Tag
	conf Confidence

	origLang   bool
	origReg    bool
	regDist    uint8
	origScript bool
	parentDist uint8 
}

func (m *bestMatch) update(have *haveTag, tag Tag, maxScript scriptID, maxRegion regionID) {

	c := have.conf
	if c < m.conf {
		return
	}
	if have.maxScript != maxScript {

		if Low < m.conf || have.altScript != maxScript {
			return
		}
		c = Low
	} else if have.maxRegion != maxRegion {

		if High < c {
			c = High
		}
	}

	beaten := false 
	if c != m.conf {
		if c < m.conf {
			return
		}
		beaten = true
	}

	origLang := have.tag.lang == tag.lang && tag.lang != 0
	if !beaten && m.origLang != origLang {
		if m.origLang {
			return
		}
		beaten = true
	}

	origReg := have.tag.region == tag.region && tag.region != 0
	if !beaten && m.origReg != origReg {
		if m.origReg {
			return
		}
		beaten = true
	}

	regDist := regionDist(have.maxRegion, maxRegion, tag.lang)
	if !beaten && m.regDist != regDist {
		if regDist > m.regDist {
			return
		}
		beaten = true
	}

	origScript := have.tag.script == tag.script && tag.script != 0
	if !beaten && m.origScript != origScript {
		if m.origScript {
			return
		}
		beaten = true
	}

	parentDist := parentDistance(have.tag.region, tag)
	if !beaten && m.parentDist != parentDist {
		if parentDist > m.parentDist {
			return
		}
		beaten = true
	}

	if beaten {
		m.have = have
		m.want = tag
		m.conf = c
		m.origLang = origLang
		m.origReg = origReg
		m.origScript = origScript
		m.regDist = regDist
		m.parentDist = parentDist
	}
}

func parentDistance(haveRegion regionID, tag Tag) uint8 {
	p := tag.Parent()
	d := uint8(1)
	for haveRegion != p.region {
		if p.region == 0 {
			return 255
		}
		p = p.Parent()
		d++
	}
	return d
}

func regionDist(a, b regionID, lang langID) uint8 {
	if lang == _en {

		if a != _US && b != _US {
			return 2
		}
	}
	return uint8(regionDistance(a, b))
}

func regionDistance(a, b regionID) int {
	if a == b {
		return 0
	}
	p, q := regionInclusion[a], regionInclusion[b]
	if p < nRegionGroups {
		p, q = q, p
	}
	set := regionInclusionBits
	if q < nRegionGroups && set[p]&(1<<q) != 0 {
		return 1
	}
	d := 2
	for goal := set[q]; set[p]&goal == 0; p = regionInclusionNext[p] {
		d++
	}
	return d
}

func (t Tag) variants() string {
	if t.pVariant == 0 {
		return ""
	}
	return t.str[t.pVariant:t.pExt]
}

func (t Tag) variantOrPrivateTagStr() string {
	if t.pExt > 0 {
		return t.str[t.pVariant:t.pExt]
	}
	return t.str[t.pVariant:]
}

func (a Tag) equalsRest(b Tag) bool {

	return a.script == b.script && a.region == b.region && a.variantOrPrivateTagStr() == b.variantOrPrivateTagStr()
}

func isExactEquivalent(l langID) bool {
	for _, o := range notEquivalent {
		if o == l {
			return false
		}
	}
	return true
}

var notEquivalent []langID

func init() {

	for _, lm := range langAliasMap {
		tag := Tag{lang: langID(lm.from)}
		if tag, _ = tag.canonicalize(All); tag.script != 0 || tag.region != 0 {
			notEquivalent = append(notEquivalent, langID(lm.from))
		}
	}
}
