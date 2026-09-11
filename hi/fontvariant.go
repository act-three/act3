package hi

import "strings"

// These options select typographic variants.
//
// If the font in use does not provide a particular variant,
// the option has no effect.
var (
	SmallCaps     FontOption = smallCaps
	AllSmallCaps  FontOption = allSmallCaps
	PetiteCaps    FontOption = petiteCaps
	AllPetiteCaps FontOption = allPetiteCaps
	Unicase       FontOption = unicase
	TitlingCaps   FontOption = titlingCaps

	LiningNums        FontOption = liningNums
	OldstyleNums      FontOption = oldstyleNums
	ProportionalNums  FontOption = proportionalNums
	TabularNums       FontOption = tabularNums
	DiagonalFractions FontOption = diagonalFractions
	StackedFractions  FontOption = stackedFractions
	Ordinal           FontOption = ordinal
	SlashedZero       FontOption = slashedZero

	CommonLigatures          FontOption = commonLigatures
	DiscretionaryLigatures   FontOption = discretionaryLigatures
	HistoricalLigatures      FontOption = historicalLigatures
	ContextualAlternates     FontOption = contextualAlternates
	NoCommonLigatures        FontOption = noCommonLigatures
	NoDiscretionaryLigatures FontOption = noDiscretionaryLigatures
	NoHistoricalLigatures    FontOption = noHistoricalLigatures
	NoContextualAlternates   FontOption = noContextualAlternates
	NoLigatures              FontOption = noLigatures

	Subscript   FontOption = subscript
	Superscript FontOption = superscript
)

type fontVariant int

const (
	smallCaps fontVariant = iota + 1
	allSmallCaps
	petiteCaps
	allPetiteCaps
	unicase
	titlingCaps

	liningNums
	oldstyleNums
	proportionalNums
	tabularNums
	diagonalFractions
	stackedFractions
	ordinal
	slashedZero

	commonLigatures
	noCommonLigatures
	discretionaryLigatures
	noDiscretionaryLigatures
	historicalLigatures
	noHistoricalLigatures
	contextualAlternates
	noContextualAlternates
	noLigatures

	subscript
	superscript
)

func (v fontVariant) apply(env environment) environment {
	switch v {
	case smallCaps, allSmallCaps, petiteCaps, allPetiteCaps, unicase, titlingCaps:
		env.fontVariants.caps = v
	case liningNums, oldstyleNums:
		env.fontVariants.numeric[numericFigure] = v
	case proportionalNums, tabularNums:
		env.fontVariants.numeric[numericSpacing] = v
	case diagonalFractions, stackedFractions:
		env.fontVariants.numeric[numericFraction] = v
	case ordinal:
		env.fontVariants.numeric[numericOrdinal] = v
	case slashedZero:
		env.fontVariants.numeric[numericZero] = v
	case commonLigatures, noCommonLigatures:
		env.fontVariants.ligatures[ligatureCommon] = v
	case discretionaryLigatures, noDiscretionaryLigatures:
		env.fontVariants.ligatures[ligatureDiscretionary] = v
	case historicalLigatures, noHistoricalLigatures:
		env.fontVariants.ligatures[ligatureHistorical] = v
	case contextualAlternates, noContextualAlternates:
		env.fontVariants.ligatures[ligatureContextual] = v
	case noLigatures:
		env.fontVariants.ligatures = [ligatureSlots]fontVariant{
			ligatureCommon:        noCommonLigatures,
			ligatureDiscretionary: noDiscretionaryLigatures,
			ligatureHistorical:    noHistoricalLigatures,
			ligatureContextual:    noContextualAlternates,
		}
	case subscript, superscript:
		env.fontVariants.position = v
	}
	return env
}

type fontVariants struct {
	caps      fontVariant
	numeric   [numericSlots]fontVariant
	ligatures [ligatureSlots]fontVariant
	position  fontVariant
}

const (
	numericFigure = iota
	numericSpacing
	numericFraction
	numericOrdinal
	numericZero
	numericSlots
)

const (
	ligatureCommon = iota
	ligatureDiscretionary
	ligatureHistorical
	ligatureContextual
	ligatureSlots
)

func variantList(values []fontVariant) string {
	var nonempty []string
	for _, v := range values {
		if v != 0 {
			nonempty = append(nonempty, v.css())
		}
	}
	return strings.Join(nonempty, " ")
}

func (v fontVariant) css() string {
	names := [...]string{
		smallCaps:                "small-caps",
		allSmallCaps:             "all-small-caps",
		petiteCaps:               "petite-caps",
		allPetiteCaps:            "all-petite-caps",
		unicase:                  "unicase",
		titlingCaps:              "titling-caps",
		liningNums:               "lining-nums",
		oldstyleNums:             "oldstyle-nums",
		proportionalNums:         "proportional-nums",
		tabularNums:              "tabular-nums",
		diagonalFractions:        "diagonal-fractions",
		stackedFractions:         "stacked-fractions",
		ordinal:                  "ordinal",
		slashedZero:              "slashed-zero",
		commonLigatures:          "common-ligatures",
		noCommonLigatures:        "no-common-ligatures",
		discretionaryLigatures:   "discretionary-ligatures",
		noDiscretionaryLigatures: "no-discretionary-ligatures",
		historicalLigatures:      "historical-ligatures",
		noHistoricalLigatures:    "no-historical-ligatures",
		contextualAlternates:     "contextual",
		noContextualAlternates:   "no-contextual",
		noLigatures:              "none",
		subscript:                "sub",
		superscript:              "super",
	}
	if v < 0 || int(v) >= len(names) {
		return ""
	}
	return names[v]
}
