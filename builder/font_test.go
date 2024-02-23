package builder

import "testing"
import "bytes"
import "slices"
import "image"
import "image/color"

import "github.com/tinne26/ggfnt"

func TestBasicFontBuild(t *testing.T) {
	// see that empty font build results in ErrBuildNoGlyphs
	builder := New()
	_, err := builder.Build()
	if err != ErrBuildNoGlyphs {
		if err == nil {
			t.Fatalf("expected ErrBuildNoGlyphs on emptyFontBuilder.Build(), but got 'nil'")
		} else {
			t.Fatalf("expected ErrBuildNoGlyphs on emptyFontBuilder.Build(), but got '%s'", err)
		}
	}

	// header changes and tests and confirmations
	builderFontID := builder.GetFontID()
	if len(builder.GetFontIDStr()) != 16 {
		t.Fatalf("expected font ID string to be 16 characters long, got '%s'", builder.GetFontIDStr())
	}
	if builder.GetVersionStr() != "v0.01" {
		t.Fatalf("expected default FontBuilder version to be v0.01, got %s", builder.GetVersionStr())
	}
	buildDateFirst := builder.GetFirstVerDate()
	buildDateMajor := builder.GetMajorVerDate()
	buildDateMinor := builder.GetMinorVerDate()
	if !buildDateFirst.IsComplete() {
		t.Fatalf("expected default FontBuilder date to be complete, got %s", buildDateFirst.String())
	}
	if buildDateFirst != buildDateMajor || buildDateMajor != buildDateMinor {
		t.Fatalf(
			"expected default FontBuilder dates to be the same, but got %s, %s, %s",
			buildDateFirst.String(), buildDateMajor.String(), buildDateMinor.String(),
		)
	}

	builder.RaiseMinorVersion()
	if builder.GetVersionStr() != "v0.02" {
		t.Fatalf("expected FontBuilder version to be v0.02, got %s", builder.GetVersionStr())
	}

	// add one glyph to the font
	mask := image.NewAlpha(image.Rect(0, -4, 3, 0)) // T
	mask.SetAlpha(0, -4, color.Alpha{255})
	mask.SetAlpha(1, -4, color.Alpha{255})
	mask.SetAlpha(2, -4, color.Alpha{255})
	mask.SetAlpha(1, -3, color.Alpha{255})
	mask.SetAlpha(1, -2, color.Alpha{255})
	mask.SetAlpha(1, -1, color.Alpha{255})
	_, err = builder.AddGlyph(mask)
	if err != nil {
		t.Fatalf("unexpected FontBuilder.AddGlyph() error: %s", err)
	}

	// check that the font is built without errors
	font, err := builder.Build()
	if err != nil {
		t.Fatalf("unexpected FontBuilder.Build() error: %s", err)
	}

	// compare built font header data with previous values
	if font.Header().FormatVersion() != ggfnt.FormatVersion {
		t.Fatalf("expected font version %d, got %d instead", ggfnt.FormatVersion, font.Header().FormatVersion())
	}
	if font.Header().ID() != builderFontID {
		t.Fatalf("expected font ID %016X, got %016X instead", builderFontID, font.Header().ID())
	}
	if font.Header().VersionMajor() != 0 {
		t.Fatalf("expected major version %d, got %d instead", 0, font.Header().VersionMajor())
	}
	if font.Header().VersionMinor() != 2 {
		t.Fatalf("expected major version %d, got %d instead", 2, font.Header().VersionMinor())
	}
	date := font.Header().FirstVersionDate()
	if date != buildDateFirst {
		t.Fatalf("expected first version date '%s', got '%s' instead", buildDateFirst.String(), date.String())
	}
	date = font.Header().MajorVersionDate()
	if date != buildDateMajor {
		t.Fatalf("expected major version date '%s', got '%s' instead", buildDateMajor.String(), date.String())
	}
	date = font.Header().MinorVersionDate()
	if date != buildDateMinor {
		t.Fatalf("expected minor version date '%s', got '%s' instead", buildDateMinor.String(), date.String())
	}
	info := font.Header().Name()
	if info != fontBuilderDefaultFontName {
		t.Fatalf("expected font name '%s', got '%s' instead", fontBuilderDefaultFontName, info)
	}
	info = font.Header().Family()
	if info != fontBuilderDefaultFontName {
		t.Fatalf("expected font family '%s', got '%s' instead", fontBuilderDefaultFontName, info)
	}
	info = font.Header().Author()
	if info != fontBuilderDefaultFontAuthor {
		t.Fatalf("expected font author '%s', got '%s' instead", fontBuilderDefaultFontAuthor, info)
	}
	info = font.Header().About()
	if info != fontBuilderDefaultFontAbout {
		t.Fatalf("expected font about '%s', got '%s' instead", fontBuilderDefaultFontAbout, info)
	}

	// check num glyphs
	if font.Metrics().NumGlyphs() != 1 {
		t.Fatalf("expected NumGlyphs() to return %d, got %d instead", 1, font.Metrics().NumGlyphs())
	}

	// export font and parse it again to see that the values are consistent
	var buffer bytes.Buffer
	err = font.Export(&buffer)
	if err != nil {
		t.Fatalf("unexpected Font.Export() error: %s", err)
	}

	reFont, err := ggfnt.Parse(&buffer)
	if err != nil {
		t.Fatalf("unexpected Parse() error: %s", err)
	}
	//reFont := internal.Font(*ggfont)
	if !slices.Equal(reFont.Data, font.Data) {
		t.Fatalf("after exporting and re-parsing, font data changed:\n>> (original build) %v\n>> (export+parse) %v", font.Data, reFont.Data)
	}

	// check offsets consistency
	if reFont.OffsetToMetrics != font.OffsetToMetrics {
		t.Fatalf("after exporting and re-parsing, offset to metrics is %d (expected %d)", reFont.OffsetToMetrics, font.OffsetToMetrics)
	}
	if reFont.OffsetToColorSections != font.OffsetToColorSections {
		t.Fatalf("after exporting and re-parsing, offset to color sections is %d (expected %d)", reFont.OffsetToColorSections, font.OffsetToColorSections)
	}
	if reFont.OffsetToColorSectionNames != font.OffsetToColorSectionNames {
		t.Fatalf("after exporting and re-parsing, offset to color section names is %d (expected %d)", reFont.OffsetToColorSectionNames, font.OffsetToColorSectionNames)
	}
	if reFont.OffsetToGlyphNames != font.OffsetToGlyphNames {
		t.Fatalf("after exporting and re-parsing, offset to glyph names is %d (expected %d)", reFont.OffsetToGlyphNames, font.OffsetToGlyphNames)
	}
	if reFont.OffsetToGlyphMasks != font.OffsetToGlyphMasks {
		t.Fatalf("after exporting and re-parsing, offset to glyph masks is %d (expected %d)", reFont.OffsetToGlyphMasks, font.OffsetToGlyphMasks)
	}
	if reFont.OffsetToWords != font.OffsetToWords {
		t.Fatalf("after exporting and re-parsing, offset to words is %d (expected %d)", reFont.OffsetToWords, font.OffsetToWords)
	}
	if reFont.OffsetToSettingNames != font.OffsetToSettingNames {
		t.Fatalf("after exporting and re-parsing, offset to setting names is %d (expected %d)", reFont.OffsetToSettingNames, font.OffsetToSettingNames)
	}
	if reFont.OffsetToSettingDefinitions != font.OffsetToSettingDefinitions {
		t.Fatalf("after exporting and re-parsing, offset to setting definitions is %d (expected %d)", reFont.OffsetToSettingDefinitions, font.OffsetToSettingDefinitions)
	}
	if reFont.OffsetToMappingSwitches != font.OffsetToMappingSwitches {
		t.Fatalf("after exporting and re-parsing, offset to mapping switches is %d (expected %d)", reFont.OffsetToMappingSwitches, font.OffsetToMappingSwitches)
	}
	if reFont.OffsetToMapping != font.OffsetToMapping {
		t.Fatalf("after exporting and re-parsing, offset to mappings is %d (expected %d)", reFont.OffsetToMapping, font.OffsetToMapping)
	}
	if reFont.OffsetToRewriteConditions != font.OffsetToRewriteConditions {
		t.Fatalf("after exporting and re-parsing, offset to rewrite conditions is %d (expected %d)", reFont.OffsetToRewriteConditions, font.OffsetToRewriteConditions)
	}
	if reFont.OffsetToGlyphRewrites != font.OffsetToGlyphRewrites {
		t.Fatalf("after exporting and re-parsing, offset to glyph rewrites is %d (expected %d)", reFont.OffsetToGlyphRewrites, font.OffsetToGlyphRewrites)
	}
	if reFont.OffsetToUtf8Rewrites != font.OffsetToUtf8Rewrites {
		t.Fatalf("after exporting and re-parsing, offset to UTF8 rewrites is %d (expected %d)", reFont.OffsetToUtf8Rewrites, font.OffsetToUtf8Rewrites)
	}
	if reFont.OffsetToHorzKernings != font.OffsetToHorzKernings {
		t.Fatalf("after exporting and re-parsing, offset to horz kernings is %d (expected %d)", reFont.OffsetToHorzKernings, font.OffsetToHorzKernings)
	}
	if reFont.OffsetToVertKernings != font.OffsetToVertKernings {
		t.Fatalf("after exporting and re-parsing, offset to vert kernings is %d (expected %d)", reFont.OffsetToVertKernings, font.OffsetToVertKernings)
	}
}

func TestExpectedParsing(t *testing.T) {
	

	// TODO: we pass t to a sub-testing function for parsing
	testingParseFontWithoutErrors(t, nil)
}

func testingParseFontWithoutErrors(t *testing.T, data []byte) {
	// ...
}
