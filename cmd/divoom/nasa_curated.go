package main

// nasaCuratedDates is the rotation pool for the NASA APOD scene. Each
// push picks one date at random and bakes that day's APOD into the bg.
//
// Curation criteria (in priority order):
//  1. Iconic / widely-shared deep-field, planetary, or eclipse imagery
//     that reads cleanly at 760×540 (the bg's image slot).
//  2. media_type == "image" — videos and animations don't bake.
//  3. Avoid charts, diagrams, comet light-curves, and other technical
//     plots that look anemic on the wall display.
//
// The bake retries with another random pick (up to nasaPickAttempts)
// if APOD returns a non-image media_type for the chosen date, so a
// dud date here costs only an extra HTTP roundtrip — but better to
// keep the pool to known-good entries.
//
// Format: "YYYY-MM-DD". APOD's archive begins 1995-06-16.
var nasaCuratedDates = []string{
	// JWST major releases — high-saturation, strong-composition press
	// images that wall-print exceptionally well.
	"2022-07-12", // SMACS 0723 first deep field
	"2022-07-13", // Carina Nebula "Cosmic Cliffs"
	"2022-07-18", // Stephan's Quintet (JWST)
	"2022-07-20", // Jupiter and ring in infrared (JWST)
	"2022-07-22", // M74 grand-design spiral (JWST)
	"2022-09-05", // Carina cliffs revisited (JWST)
	"2022-10-13", // WR 140 dust shells (JWST)
	"2022-10-20", // Pillars of Creation (JWST)
	"2022-12-06", // M16 Eagle Nebula pillar (JWST)
	"2023-01-18", // MACS0647 gravitational lens (JWST)
	"2023-02-18", // NGC 1365 barred spiral (JWST)
	"2023-07-13", // JWST 1st anniversary — Rho Ophiuchi
	"2023-07-25", // Eagle Nebula pillars multi-wavelength
	"2023-08-14", // Ring Nebula M57 (JWST)
	"2023-10-10", // Orion Nebula (JWST)
	"2023-12-14", // Cassiopeia A supernova remnant (JWST)
	"2023-12-29", // Uranus and rings (JWST)
	"2024-04-15", // M82 Cigar starburst (JWST)
	"2024-04-16", // Vela supernova remnant (JWST)
	"2024-07-30", // Penguin & Egg (Arp 142, JWST 2nd anniversary)
	"2024-11-26", // Sombrero Galaxy (JWST + Hubble)

	// Hubble milestones and signature deep images.
	"1996-01-24", // Hubble Deep Field (original release)
	"2004-03-09", // Hubble Ultra Deep Field
	"2012-10-14", // Hubble Extreme Deep Field (XDF)
	"2015-01-07", // Hubble 25th: Pillars of Creation revisited
	"2015-04-23", // Hubble 25th anniversary — Westerlund 2
	"2007-02-18", // Pillars of Creation (classic Hubble)
	"2020-12-06", // Pillars of star creation (Hubble)
	"2019-06-23", // Carina Nebula 48-frame Hubble panorama
	"2005-04-28", // M51 sharpest Whirlpool (ACS)
	"2019-03-29", // M104 Sombrero (Hubble)
	"2003-10-08", // Sombrero Galaxy from HST
	"2022-04-23", // M104 Messier 104 (Hubble, 10+ hours)

	// Eclipses — total solar, annular.
	"2017-08-22", // 2017 US total solar eclipse — corona
	"2017-08-25", // 2017 eclipse — diamond ring through clouds
	"2017-08-23", // ISS transit during 2017 eclipse
	"2024-04-07", // Total solar eclipse over Wyoming
	"2024-04-11", // "Eclipse in Seven" composite
	"2024-04-13", // Palm-tree partial eclipse silhouette
	"2024-04-17", // Total eclipse with comets
	"2024-04-02", // Detailed eclipse corona
	"2012-05-22", // Annular eclipse over New Mexico (silhouette)
	"2024-10-08", // Annular eclipse over Patagonia

	// Aurorae.
	"2024-05-12", // Red aurora over Poland (May 2024 storm)
	"2024-05-20", // Aurora Dome Sky (May 2024 storm)
	"2024-05-22", // Green aurora over Sweden
	"2023-04-19", // Auroral storm over Lapland
	"2024-10-16", // Colorful aurora over New Zealand
	"2023-07-30", // Spiral aurora Icelandic Divide
	"2015-07-04", // Aurora Australis from South Pole Station

	// Solar system flagships — Saturn, Jupiter, Mars, Pluto, Venus.
	"2017-09-16", // Cassini's Final Image (Saturn farewell)
	"2017-09-26", // Cassini's Last Ring Portrait
	"2017-04-30", // Cassini's first dive between rings
	"2013-07-22", // Earth and Moon from Saturn ("Day Earth Smiled")
	"2019-05-08", // Jupiter Marble from Juno (Great Red Spot)
	"2015-07-14", // New Horizons: Pluto and Charon
	"2015-07-15", // Pluto Resolved (heart in detail)
	"2015-07-16", // 50 Miles on Pluto
	"2015-09-18", // A Plutonian Landscape (blue twilight)
	"2015-11-14", // Wright Mons (Pluto cryovolcano)
	"2015-12-14", // Pluto: From Mountains to Plains
	"2021-02-19", // Mars Perseverance Sol 0
	"2021-02-26", // Perseverance: 360 from Jezero Crater
	"2021-04-06", // Perseverance selfie with Ingenuity
	"2004-06-09", // Venus transit at sunrise (2004)
	"2004-06-10", // Venus at the edge (atmosphere arc)
	"2012-06-07", // Venus transit 2012 H-alpha
	"2012-06-13", // Venus transit over Baltic with green flash

	// Iconic Earth-and-spacecraft views.
	"2020-02-14", // Pale Blue Dot remastered (Voyager 30th)
	"2018-12-24", // Earthrise 1 remastered (Apollo 8)
	"2008-12-24", // Earthrise (Apollo 8 anniversary)
	"2005-12-24", // Earthrise
	"2015-09-06", // Earthrise (LRO perspective)

	// Black holes / EHT.
	"2019-04-11", // First EHT horizon image (M87)
	"2019-04-27", // M87 galaxy, jet, and black hole
	"2022-05-01", // First EHT image of Sagittarius A*
	"2022-05-13", // The Milky Way's Black Hole
	"2021-03-31", // M87 in polarized light

	// Famous nebulae — Hubble & ground.
	"1995-11-06", // Pillars of Creation (original 1995 release)
	"2010-04-26", // M16 Pillars (Hubble)
	"2017-12-27", // Horsehead Nebula (CFHT)
	"2022-12-29", // Horsehead skyscape
	"2019-12-17", // Horsehead narrowband+broadband
	"2024-11-25", // Horsehead from Chilescope
	"2024-11-04", // Orion Nebula M42 deep
	"2019-10-30", // Inside the Orion Nebula
	"2021-06-29", // M42 sharpest (Hubble)
	"2019-02-13", // Helix Nebula in H and O
	"2014-10-12", // Helix Nebula (Eye of God)
	"2020-08-23", // Helix Nebula deep
	"2023-01-15", // Crab Nebula (Hubble three-color)
	"2005-12-02", // Crab Nebula Hubble mosaic
	"2024-01-07", // Cat's Eye in optical and X-ray

	// Comets and meteors.
	"2020-07-09", // Comet NEOWISE with noctilucent clouds
	"2020-07-14", // Comet NEOWISE over Stonehenge
	"2020-07-15", // NEOWISE over the Swiss Alps
	"2020-07-23", // Fairytale NEOWISE
	"1997-03-25", // Comet Hale-Bopp — brightest of the century
	"1997-04-01", // Hale-Bopp and Andromeda
	"1997-04-29", // Hale-Bopp and Orion
	"2024-10-14", // Comet Tsuchinshan-ATLAS
	"2024-10-19", // Comet Tsuchinshan-ATLAS
	"2024-11-06", // Comet Tsuchinshan-ATLAS late tail
	"2007-01-22", // Comet McNaught over Chile

	// Galaxies — wide and dramatic.
	"2025-02-21", // Andromeda largest Hubble photomosaic
	"2023-03-22", // Andromeda 15-hour deep mosaic
	"2019-09-05", // Large Magellanic Cloud deep colour
	"2023-03-07", // LMC deep field (1060 hours)
	"2024-03-08", // Tarantula Zone in LMC

	// Milky Way / wide-field landscape astrophotography.
	"2022-07-19", // Pleiades over Half Dome (Yosemite)
	"2017-08-21", // Milky Way over Chilean volcanoes
	"2015-11-01", // Milky Way over Monument Valley
	"2008-07-13", // Dark sky over Death Valley (360 pano)
	"2021-11-01", // Waterfall and the Milky Way
	"2021-01-13", // Arches across an Arctic sky

	// ISS and spacecraft silhouettes.
	"2022-09-03", // ISS crosses both Moon and Sun
	"2019-10-28", // ISS crosses a spotless Sun
	"2010-05-23", // ISS and Atlantis transit the Sun together

	// Historic Apollo frames.
	"2019-07-20", // Apollo 11 Landing Panorama (50th)
	"2017-07-22", // Aldrin "Catching Some Sun" (solar wind)
	"2023-07-22", // Armstrong's lunar selfie (unwrapped visor)
}
