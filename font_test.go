package ggfnt

import "testing"
import "bytes"
import "slices"
import "image"
import "image/color"

func TestBasicFontBuild(t *testing.T) {
	// see that empty font build results in ErrBuildNoGlyphs
	builder := NewFontBuilder()
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
	if font.Header().FormatVersion() != FormatVersion {
		t.Fatalf("expected font version %d, got %d instead", FormatVersion, font.Header().FormatVersion())
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

	reFont, err := Parse(&buffer)
	if err != nil {
		t.Fatalf("unexpected Parse() error: %s", err)
	}
	if !slices.Equal(reFont.data, font.data) {
		t.Fatalf("after exporting and re-parsing, font data changed:\n>> (original build) %v\n>> (export+parse) %v", font.data, reFont.data)
	}

	// check offsets consistency
	if reFont.offsetToMetrics != font.offsetToMetrics {
		t.Fatalf("after exporting and re-parsing, offset to metrics is %d (expected %d)", reFont.offsetToMetrics, font.offsetToMetrics)
	}
	if reFont.offsetToGlyphNames != font.offsetToGlyphNames {
		t.Fatalf("after exporting and re-parsing, offset to glyph names is %d (expected %d)", reFont.offsetToGlyphNames, font.offsetToGlyphNames)
	}
	if reFont.offsetToGlyphMasks != font.offsetToGlyphMasks {
		t.Fatalf("after exporting and re-parsing, offset to glyph masks is %d (expected %d)", reFont.offsetToGlyphMasks, font.offsetToGlyphMasks)
	}
	if reFont.offsetToColorSections != font.offsetToColorSections {
		t.Fatalf("after exporting and re-parsing, offset to color sections is %d (expected %d)", reFont.offsetToColorSections, font.offsetToColorSections)
	}
	if reFont.offsetToColorSectionNames != font.offsetToColorSectionNames {
		t.Fatalf("after exporting and re-parsing, offset to color section names is %d (expected %d)", reFont.offsetToColorSectionNames, font.offsetToColorSectionNames)
	}
	if reFont.offsetToVariables != font.offsetToVariables {
		t.Fatalf("after exporting and re-parsing, offset to variables is %d (expected %d)", reFont.offsetToVariables, font.offsetToVariables)
	}
	if reFont.offsetToMappingModes != font.offsetToMappingModes {
		t.Fatalf("after exporting and re-parsing, offset to mapping modes is %d (expected %d)", reFont.offsetToMappingModes, font.offsetToMappingModes)
	}
	// TODO: offsetsToFastMapTables (iter)
	if reFont.offsetToMainMappings != font.offsetToMainMappings {
		t.Fatalf("after exporting and re-parsing, offset to main mappings is %d (expected %d)", reFont.offsetToMainMappings, font.offsetToMainMappings)
	}
	if reFont.offsetToHorzKernings != font.offsetToHorzKernings {
		t.Fatalf("after exporting and re-parsing, offset to horz kernings is %d (expected %d)", reFont.offsetToHorzKernings, font.offsetToHorzKernings)
	}
	if reFont.offsetToVertKernings != font.offsetToVertKernings {
		t.Fatalf("after exporting and re-parsing, offset to vert kernings is %d (expected %d)", reFont.offsetToVertKernings, font.offsetToVertKernings)
	}
}

func TestExpectedParsing(t *testing.T) {
	

	// TODO: we pass t to a sub-testing function for parsing
	testingParseFontWithoutErrors(t, nil)
}

func testingParseFontWithoutErrors(t *testing.T, data []byte) {
	// ...
}
