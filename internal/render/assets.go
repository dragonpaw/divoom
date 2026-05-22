// Embedded raster assets used as shape masks by the render package.
//
// starfleet-delta.png is derived from File:Delta-shield.svg on Wikimedia
// Commons:
//
//	https://commons.wikimedia.org/wiki/File:Delta-shield.svg
//	https://upload.wikimedia.org/wikipedia/commons/9/90/Delta-shield.svg
//
// The source SVG is in the public domain (PD-shape: "This image of simple
// geometry is ineligible for copyright and therefore in the public
// domain"). Only the silhouette is used here — the colour is overpainted
// at render time.
//
// To regenerate the PNG from the source SVG:
//
//	curl -o /tmp/delta-shield.svg \
//	  https://upload.wikimedia.org/wikipedia/commons/9/90/Delta-shield.svg
//	rsvg-convert -h 200 -o internal/render/assets/starfleet-delta.png \
//	  /tmp/delta-shield.svg
//
// devil.png is the "imp" emoji (👿, U+1F47F) from Twemoji, Twitter's
// open-source emoji set:
//
//	https://github.com/twitter/twemoji
//	https://raw.githubusercontent.com/twitter/twemoji/master/assets/svg/1f47f.svg
//
// Licensed under CC-BY 4.0 (graphics) — Copyright 2020 Twitter, Inc and
// other contributors. We only use the silhouette mask; every opaque pixel
// is overpainted in GruvBgDarker at render time, so the original Twemoji
// colours are discarded. Chosen for the devil scene because the horned
// imp head reads as the cover of Bierce's Devil's Dictionary.
//
// To regenerate the PNG from the source SVG:
//
//	curl -o /tmp/1f47f.svg \
//	  https://raw.githubusercontent.com/twitter/twemoji/master/assets/svg/1f47f.svg
//	rsvg-convert -w 200 -h 200 -o internal/render/assets/devil.png \
//	  /tmp/1f47f.svg
//
// buddha.png is the "person in lotus position" emoji (🧘, U+1F9D8) from
// Twemoji, Twitter's open-source emoji set:
//
//	https://github.com/twitter/twemoji
//	https://raw.githubusercontent.com/twitter/twemoji/master/assets/svg/1f9d8.svg
//
// Licensed under CC-BY 4.0 (graphics) — Copyright 2020 Twitter, Inc and
// other contributors. We only use the silhouette mask; every opaque pixel
// is overpainted in GruvBgDarker at render time, so the original Twemoji
// colours are discarded.
//
// To regenerate the PNG from the source SVG:
//
//	curl -o /tmp/1f9d8.svg \
//	  https://raw.githubusercontent.com/twitter/twemoji/master/assets/svg/1f9d8.svg
//	rsvg-convert -w 200 -h 200 -o internal/render/assets/buddha.png \
//	  /tmp/1f9d8.svg
//
// The weather/*.png masks are rasterised from Erik Flowers' Weather Icons
// (https://erikflowers.github.io/weather-icons/), licensed under the SIL
// Open Font License 1.1. Each outlook maps to one icon in the set:
//
//	clear    -> wi-day-sunny.svg
//	cloudy   -> wi-cloud.svg
//	overcast -> wi-cloudy.svg
//	rain     -> wi-rain.svg
//	drizzle  -> wi-showers.svg
//	snow     -> wi-snow.svg
//	fog      -> wi-fog.svg
//	thunder  -> wi-thunderstorm.svg
//	smoke    -> wi-smoke.svg
//
// question.png is the "white question mark ornament" emoji (❔, U+2754)
// from Twemoji, Twitter's open-source emoji set:
//
//	https://github.com/twitter/twemoji
//	https://raw.githubusercontent.com/twitter/twemoji/master/assets/svg/2754.svg
//
// Licensed under CC-BY 4.0 (graphics) — Copyright 2020 Twitter, Inc and
// other contributors. We only use the silhouette mask; every opaque pixel
// is overpainted in GruvBgDarker at render time, so the original Twemoji
// colours are discarded. Used by the did-you-know scene as a chunky
// typographic "?" glyph in place of the earlier hand-rasterised version.
//
// To regenerate the PNG from the source SVG:
//
//	curl -o /tmp/2754.svg \
//	  https://raw.githubusercontent.com/twitter/twemoji/master/assets/svg/2754.svg
//	rsvg-convert -w 200 -h 200 -o internal/render/assets/question.png /tmp/2754.svg
//
// til.png is the "light-bulb" solid icon from Heroicons, Tailwind Labs'
// open-source icon set:
//
//	https://github.com/tailwindlabs/heroicons
//	https://raw.githubusercontent.com/tailwindlabs/heroicons/master/src/24/solid/light-bulb.svg
//
// Licensed MIT — Copyright Tailwind Labs, Inc. We only use the silhouette
// mask; every opaque pixel is overpainted at render time. Used by the TIL
// (r/todayilearned) scene as the "new idea / fact" glyph in the bottom-
// right corner.
//
// To regenerate the PNG from the source SVG:
//
//	curl -o /tmp/til-lightbulb.svg \
//	  https://raw.githubusercontent.com/tailwindlabs/heroicons/master/src/24/solid/light-bulb.svg
//	rsvg-convert -w 200 -h 200 -o internal/render/assets/til.png /tmp/til-lightbulb.svg
//
// hazard.png is the "warning sign" emoji (⚠, U+26A0) from Twemoji,
// Twitter's open-source emoji set:
//
//	https://github.com/twitter/twemoji
//	https://raw.githubusercontent.com/twitter/twemoji/master/assets/svg/26a0.svg
//
// Licensed under CC-BY 4.0 (graphics) — Copyright 2020 Twitter, Inc and
// other contributors. We only use the silhouette mask; every opaque pixel
// is overpainted at render time. Used by the weather scene when an active
// NWS alert is in effect for the configured point.
//
// To regenerate the PNG from the source SVG:
//
//	curl -o /tmp/26a0.svg \
//	  https://raw.githubusercontent.com/twitter/twemoji/master/assets/svg/26a0.svg
//	rsvg-convert -w 200 -h 200 -o internal/render/assets/hazard.png /tmp/26a0.svg
//
// To regenerate (one outlook shown):
//
//	curl -o /tmp/wi-day-sunny.svg \
//	  https://raw.githubusercontent.com/erikflowers/weather-icons/master/svg/wi-day-sunny.svg
//	rsvg-convert -w 200 -h 200 \
//	  -o internal/render/assets/weather/clear.png /tmp/wi-day-sunny.svg
package render

import _ "embed"

//go:embed assets/starfleet-delta.png
var starfleetDeltaPNG []byte

//go:embed assets/buddha.png
var buddhaPNG []byte

//go:embed assets/devil.png
var devilPNG []byte

//go:embed assets/weather/clear.png
var weatherClearPNG []byte

//go:embed assets/weather/cloudy.png
var weatherCloudyPNG []byte

//go:embed assets/weather/overcast.png
var weatherOvercastPNG []byte

//go:embed assets/weather/rain.png
var weatherRainPNG []byte

//go:embed assets/weather/drizzle.png
var weatherDrizzlePNG []byte

//go:embed assets/weather/snow.png
var weatherSnowPNG []byte

//go:embed assets/weather/fog.png
var weatherFogPNG []byte

//go:embed assets/weather/thunder.png
var weatherThunderPNG []byte

//go:embed assets/weather/smoke.png
var weatherSmokePNG []byte

//go:embed assets/hazard.png
var hazardPNG []byte

//go:embed assets/question.png
var questionPNG []byte

//go:embed assets/til.png
var tilPNG []byte
