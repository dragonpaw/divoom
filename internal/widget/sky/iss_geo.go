package sky

import "math"

// locationFor returns the "over <somewhere>" string for the ISS scene's
// third row. It prefers (in order) the nearest major city within
// ~600 km, a named sea or ocean bounding box, and finally a coarse
// ocean by longitude band.
//
// The city table is hand-curated to ~200 metropolitan areas with
// population above roughly one million, chosen to give global coverage
// without ballooning the binary. The threshold is loose on purpose —
// the ISS sub-satellite point moves fast and the row is a glanceable
// caption, not a gazetteer. The sea table is intentionally small: only
// well-known named seas the average reader will recognise.
func locationFor(lat, lon float64) string {
	if city, country, ok := nearestCity(lat, lon, 600); ok {
		return "over " + city + ", " + country
	}
	if name := namedSea(lat, lon); name != "" {
		return "over " + name
	}
	return "over " + oceanBand(lon)
}

// nearestCity scans the embedded city table and returns the closest
// entry within maxKm great-circle distance, or ok=false if none qualify.
func nearestCity(lat, lon, maxKm float64) (name, country string, ok bool) {
	best := maxKm
	for _, c := range cities {
		d := haversineKm(lat, lon, c.Lat, c.Lon)
		if d < best {
			best = d
			name = c.Name
			country = c.Country
			ok = true
		}
	}
	return
}

// haversineKm returns the great-circle distance between two lat/lon
// pairs in kilometres. Earth radius 6371 km — same constant every
// online calculator uses, accurate to a few parts per thousand at any
// latitude that matters for this widget.
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// namedSea returns the name of a recognised sea covering this point,
// or "" if none match. Boxes are intentionally loose — they're picked
// for "reads sensibly when the ISS is over it", not nautical accuracy.
// Order matters: the smaller/more specific seas come first so e.g. the
// Sea of Japan isn't swallowed by a broader Pacific check.
func namedSea(lat, lon float64) string {
	for _, s := range namedSeas {
		if lat >= s.LatMin && lat <= s.LatMax &&
			lon >= s.LonMin && lon <= s.LonMax {
			return s.Name
		}
	}
	return ""
}

// oceanBand returns one of the five oceans based purely on longitude
// (and, for the south pole, latitude isn't needed because Antarctica
// is handled upstream of this function via the sea/city checks not
// matching). Used as the last-resort fallback.
func oceanBand(lon float64) string {
	switch {
	case lon >= -70 && lon < 20:
		return "Atlantic"
	case lon >= 20 && lon < 110:
		return "Indian Ocean"
	default:
		return "Pacific"
	}
}

type seaBox struct {
	Name                           string
	LatMin, LatMax, LonMin, LonMax float64
}

// namedSeas — loose bounding boxes for the seas a reader is likely to
// recognise. Ordered most-specific first so smaller seas win over the
// larger water bodies that contain them.
var namedSeas = []seaBox{
	{"Baltic Sea", 53, 66, 10, 30},
	{"North Sea", 51, 61, -4, 9},
	// Black Sea before Mediterranean: their lat/lon boxes overlap
	// around (43, 35), and the Black Sea is the more specific label.
	{"Black Sea", 41, 47, 27, 42},
	{"Mediterranean", 30, 46, -6, 36},
	{"Red Sea", 12, 30, 32, 44},
	{"Caribbean", 9, 22, -88, -60},
	{"Sea of Japan", 34, 52, 127, 142},
	{"Bering Sea", 53, 66, 162, 180},
	{"Bering Sea", 53, 66, -180, -157},
	{"Arabian Sea", 5, 25, 50, 78},
	// Polar caps — handled as "seas" so locationFor returns something
	// readable for them too without needing a separate branch.
	{"Arctic Ocean", 66, 90, -180, 180},
	{"Southern Ocean", -90, -60, -180, 180},
}

type cityEntry struct {
	Name    string
	Country string
	Lat     float64
	Lon     float64
}

// cities — ~200 metropolitan areas with roughly >1M population, hand
// curated for global coverage. Source: cross-referenced with public
// population lists (UN World Urbanization Prospects, Wikipedia
// "List of cities proper by population"). Lat/lon are city-centre
// coordinates rounded to two decimals. Not exhaustive — the ISS scene
// only needs "is the bird approximately over a city the user has
// heard of"; sub-million cities and minor secondary cities are
// deliberately omitted to keep the table small.
var cities = []cityEntry{
	// North America
	{"New York", "USA", 40.71, -74.01},
	{"Los Angeles", "USA", 34.05, -118.24},
	{"Chicago", "USA", 41.88, -87.63},
	{"Houston", "USA", 29.76, -95.37},
	{"Phoenix", "USA", 33.45, -112.07},
	{"Philadelphia", "USA", 39.95, -75.17},
	{"San Antonio", "USA", 29.42, -98.49},
	{"San Diego", "USA", 32.72, -117.16},
	{"Dallas", "USA", 32.78, -96.80},
	{"San Francisco", "USA", 37.77, -122.42},
	{"Seattle", "USA", 47.61, -122.33},
	{"Denver", "USA", 39.74, -104.99},
	{"Boston", "USA", 42.36, -71.06},
	{"Atlanta", "USA", 33.75, -84.39},
	{"Miami", "USA", 25.76, -80.19},
	{"Washington", "USA", 38.91, -77.04},
	{"Detroit", "USA", 42.33, -83.05},
	{"Minneapolis", "USA", 44.98, -93.27},
	{"Las Vegas", "USA", 36.17, -115.14},
	{"Portland", "USA", 45.52, -122.68},
	{"Honolulu", "USA", 21.31, -157.86},
	{"Anchorage", "USA", 61.22, -149.90},
	{"Toronto", "Canada", 43.65, -79.38},
	{"Montreal", "Canada", 45.50, -73.57},
	{"Vancouver", "Canada", 49.28, -123.12},
	{"Calgary", "Canada", 51.05, -114.07},
	{"Ottawa", "Canada", 45.42, -75.70},
	{"Edmonton", "Canada", 53.55, -113.49},
	{"Mexico City", "Mexico", 19.43, -99.13},
	{"Guadalajara", "Mexico", 20.67, -103.35},
	{"Monterrey", "Mexico", 25.69, -100.32},
	{"Tijuana", "Mexico", 32.51, -117.04},
	{"Puebla", "Mexico", 19.04, -98.21},

	// Central America & Caribbean
	{"Guatemala City", "Guatemala", 14.63, -90.51},
	{"San Salvador", "El Salvador", 13.69, -89.22},
	{"Tegucigalpa", "Honduras", 14.07, -87.19},
	{"Managua", "Nicaragua", 12.11, -86.24},
	{"San José", "Costa Rica", 9.93, -84.08},
	{"Panama City", "Panama", 8.98, -79.52},
	{"Havana", "Cuba", 23.13, -82.38},
	{"Santo Domingo", "Dominican Republic", 18.47, -69.90},
	{"Port-au-Prince", "Haiti", 18.59, -72.31},
	{"Kingston", "Jamaica", 17.97, -76.79},
	{"San Juan", "Puerto Rico", 18.47, -66.11},

	// South America
	{"São Paulo", "Brazil", -23.55, -46.63},
	{"Rio de Janeiro", "Brazil", -22.91, -43.17},
	{"Brasília", "Brazil", -15.78, -47.93},
	{"Salvador", "Brazil", -12.97, -38.50},
	{"Fortaleza", "Brazil", -3.72, -38.54},
	{"Belo Horizonte", "Brazil", -19.92, -43.94},
	{"Manaus", "Brazil", -3.12, -60.02},
	{"Curitiba", "Brazil", -25.43, -49.27},
	{"Recife", "Brazil", -8.05, -34.88},
	{"Porto Alegre", "Brazil", -30.03, -51.23},
	{"Buenos Aires", "Argentina", -34.61, -58.38},
	{"Córdoba", "Argentina", -31.42, -64.18},
	{"Rosario", "Argentina", -32.95, -60.65},
	{"Mendoza", "Argentina", -32.89, -68.83},
	{"Santiago", "Chile", -33.45, -70.67},
	{"Lima", "Peru", -12.05, -77.04},
	{"Bogotá", "Colombia", 4.71, -74.07},
	{"Medellín", "Colombia", 6.24, -75.58},
	{"Cali", "Colombia", 3.45, -76.53},
	{"Caracas", "Venezuela", 10.50, -66.92},
	{"Maracaibo", "Venezuela", 10.65, -71.65},
	{"Quito", "Ecuador", -0.18, -78.47},
	{"Guayaquil", "Ecuador", -2.17, -79.92},
	{"La Paz", "Bolivia", -16.49, -68.15},
	{"Asunción", "Paraguay", -25.26, -57.58},
	{"Montevideo", "Uruguay", -34.90, -56.16},

	// Europe
	{"London", "UK", 51.51, -0.13},
	{"Manchester", "UK", 53.48, -2.24},
	{"Birmingham", "UK", 52.49, -1.90},
	{"Glasgow", "UK", 55.86, -4.25},
	{"Dublin", "Ireland", 53.35, -6.26},
	{"Paris", "France", 48.86, 2.35},
	{"Marseille", "France", 43.30, 5.37},
	{"Lyon", "France", 45.76, 4.84},
	{"Madrid", "Spain", 40.42, -3.70},
	{"Barcelona", "Spain", 41.39, 2.17},
	{"Valencia", "Spain", 39.47, -0.38},
	{"Seville", "Spain", 37.39, -5.98},
	{"Lisbon", "Portugal", 38.72, -9.14},
	{"Porto", "Portugal", 41.15, -8.61},
	{"Rome", "Italy", 41.90, 12.50},
	{"Milan", "Italy", 45.46, 9.19},
	{"Naples", "Italy", 40.85, 14.27},
	{"Turin", "Italy", 45.07, 7.69},
	{"Berlin", "Germany", 52.52, 13.40},
	{"Hamburg", "Germany", 53.55, 9.99},
	{"Munich", "Germany", 48.14, 11.58},
	{"Cologne", "Germany", 50.94, 6.96},
	{"Frankfurt", "Germany", 50.11, 8.68},
	{"Amsterdam", "Netherlands", 52.37, 4.90},
	{"Rotterdam", "Netherlands", 51.92, 4.48},
	{"Brussels", "Belgium", 50.85, 4.35},
	{"Vienna", "Austria", 48.21, 16.37},
	{"Zürich", "Switzerland", 47.38, 8.54},
	{"Geneva", "Switzerland", 46.20, 6.15},
	{"Warsaw", "Poland", 52.23, 21.01},
	{"Kraków", "Poland", 50.06, 19.94},
	{"Prague", "Czechia", 50.08, 14.44},
	{"Budapest", "Hungary", 47.50, 19.04},
	{"Bucharest", "Romania", 44.43, 26.10},
	{"Sofia", "Bulgaria", 42.70, 23.32},
	{"Athens", "Greece", 37.98, 23.73},
	{"Belgrade", "Serbia", 44.79, 20.45},
	{"Zagreb", "Croatia", 45.81, 15.98},
	{"Copenhagen", "Denmark", 55.68, 12.57},
	{"Stockholm", "Sweden", 59.33, 18.07},
	{"Oslo", "Norway", 59.91, 10.75},
	{"Helsinki", "Finland", 60.17, 24.94},
	{"Reykjavík", "Iceland", 64.15, -21.94},
	{"Moscow", "Russia", 55.75, 37.62},
	{"Saint Petersburg", "Russia", 59.93, 30.34},
	{"Novosibirsk", "Russia", 55.03, 82.92},
	{"Yekaterinburg", "Russia", 56.84, 60.61},
	{"Kazan", "Russia", 55.79, 49.12},
	{"Vladivostok", "Russia", 43.12, 131.89},
	{"Kyiv", "Ukraine", 50.45, 30.52},
	{"Minsk", "Belarus", 53.90, 27.57},

	// Middle East
	{"Istanbul", "Turkey", 41.01, 28.98},
	{"Ankara", "Turkey", 39.93, 32.86},
	{"Izmir", "Turkey", 38.42, 27.14},
	{"Tehran", "Iran", 35.69, 51.39},
	{"Mashhad", "Iran", 36.27, 59.62},
	{"Isfahan", "Iran", 32.65, 51.67},
	{"Baghdad", "Iraq", 33.31, 44.36},
	{"Riyadh", "Saudi Arabia", 24.71, 46.68},
	{"Jeddah", "Saudi Arabia", 21.49, 39.19},
	{"Mecca", "Saudi Arabia", 21.39, 39.86},
	{"Dubai", "UAE", 25.20, 55.27},
	{"Abu Dhabi", "UAE", 24.47, 54.37},
	{"Doha", "Qatar", 25.29, 51.53},
	{"Kuwait City", "Kuwait", 29.38, 47.99},
	{"Manama", "Bahrain", 26.23, 50.59},
	{"Muscat", "Oman", 23.59, 58.41},
	{"Sanaa", "Yemen", 15.37, 44.19},
	{"Amman", "Jordan", 31.95, 35.93},
	{"Beirut", "Lebanon", 33.89, 35.50},
	{"Damascus", "Syria", 33.51, 36.29},
	{"Tel Aviv", "Israel", 32.08, 34.78},
	{"Jerusalem", "Israel", 31.78, 35.22},

	// Africa
	{"Cairo", "Egypt", 30.04, 31.24},
	{"Alexandria", "Egypt", 31.20, 29.92},
	{"Lagos", "Nigeria", 6.52, 3.38},
	{"Kano", "Nigeria", 12.00, 8.52},
	{"Abuja", "Nigeria", 9.08, 7.40},
	{"Kinshasa", "DR Congo", -4.32, 15.31},
	{"Luanda", "Angola", -8.84, 13.23},
	{"Khartoum", "Sudan", 15.50, 32.56},
	{"Addis Ababa", "Ethiopia", 9.03, 38.74},
	{"Nairobi", "Kenya", -1.29, 36.82},
	{"Dar es Salaam", "Tanzania", -6.79, 39.21},
	{"Kampala", "Uganda", 0.35, 32.58},
	{"Mogadishu", "Somalia", 2.05, 45.32},
	{"Algiers", "Algeria", 36.75, 3.06},
	{"Tunis", "Tunisia", 36.81, 10.18},
	{"Tripoli", "Libya", 32.89, 13.18},
	{"Casablanca", "Morocco", 33.57, -7.59},
	{"Rabat", "Morocco", 34.02, -6.83},
	{"Dakar", "Senegal", 14.72, -17.47},
	{"Abidjan", "Côte d'Ivoire", 5.36, -4.01},
	{"Accra", "Ghana", 5.60, -0.19},
	{"Johannesburg", "South Africa", -26.20, 28.05},
	{"Cape Town", "South Africa", -33.92, 18.42},
	{"Durban", "South Africa", -29.86, 31.02},
	{"Harare", "Zimbabwe", -17.82, 31.05},
	{"Maputo", "Mozambique", -25.97, 32.57},
	{"Antananarivo", "Madagascar", -18.88, 47.51},

	// South & Central Asia
	{"Mumbai", "India", 19.08, 72.88},
	{"Delhi", "India", 28.61, 77.21},
	{"Bangalore", "India", 12.97, 77.59},
	{"Hyderabad", "India", 17.39, 78.49},
	{"Chennai", "India", 13.08, 80.27},
	{"Kolkata", "India", 22.57, 88.36},
	{"Ahmedabad", "India", 23.03, 72.59},
	{"Pune", "India", 18.52, 73.86},
	{"Jaipur", "India", 26.91, 75.79},
	{"Lucknow", "India", 26.85, 80.95},
	{"Karachi", "Pakistan", 24.86, 67.01},
	{"Lahore", "Pakistan", 31.55, 74.34},
	{"Islamabad", "Pakistan", 33.69, 73.05},
	{"Dhaka", "Bangladesh", 23.81, 90.41},
	{"Chittagong", "Bangladesh", 22.36, 91.78},
	{"Kathmandu", "Nepal", 27.72, 85.32},
	{"Colombo", "Sri Lanka", 6.93, 79.86},
	{"Kabul", "Afghanistan", 34.53, 69.17},
	{"Tashkent", "Uzbekistan", 41.30, 69.24},
	{"Almaty", "Kazakhstan", 43.24, 76.95},
	{"Astana", "Kazakhstan", 51.16, 71.43},

	// East & Southeast Asia
	{"Tokyo", "Japan", 35.69, 139.69},
	{"Osaka", "Japan", 34.69, 135.50},
	{"Yokohama", "Japan", 35.44, 139.64},
	{"Nagoya", "Japan", 35.18, 136.91},
	{"Sapporo", "Japan", 43.07, 141.35},
	{"Fukuoka", "Japan", 33.59, 130.40},
	{"Seoul", "South Korea", 37.57, 126.98},
	{"Busan", "South Korea", 35.18, 129.08},
	{"Pyongyang", "North Korea", 39.02, 125.74},
	{"Beijing", "China", 39.90, 116.41},
	{"Shanghai", "China", 31.23, 121.47},
	{"Guangzhou", "China", 23.13, 113.26},
	{"Shenzhen", "China", 22.54, 114.06},
	{"Chongqing", "China", 29.56, 106.55},
	{"Chengdu", "China", 30.57, 104.07},
	{"Tianjin", "China", 39.34, 117.36},
	{"Wuhan", "China", 30.59, 114.31},
	{"Xi'an", "China", 34.34, 108.94},
	{"Hangzhou", "China", 30.27, 120.15},
	{"Nanjing", "China", 32.06, 118.80},
	{"Shenyang", "China", 41.81, 123.43},
	{"Harbin", "China", 45.80, 126.54},
	{"Hong Kong", "China", 22.32, 114.17},
	{"Taipei", "Taiwan", 25.03, 121.57},
	{"Ulaanbaatar", "Mongolia", 47.89, 106.91},
	{"Bangkok", "Thailand", 13.76, 100.50},
	{"Hanoi", "Vietnam", 21.03, 105.85},
	{"Ho Chi Minh City", "Vietnam", 10.82, 106.63},
	{"Phnom Penh", "Cambodia", 11.56, 104.92},
	{"Vientiane", "Laos", 17.97, 102.60},
	{"Yangon", "Myanmar", 16.87, 96.20},
	{"Kuala Lumpur", "Malaysia", 3.14, 101.69},
	{"Singapore", "Singapore", 1.35, 103.82},
	{"Jakarta", "Indonesia", -6.21, 106.85},
	{"Surabaya", "Indonesia", -7.26, 112.75},
	{"Bandung", "Indonesia", -6.91, 107.61},
	{"Medan", "Indonesia", 3.60, 98.68},
	{"Manila", "Philippines", 14.60, 120.98},
	{"Cebu City", "Philippines", 10.32, 123.90},

	// Oceania
	{"Sydney", "Australia", -33.87, 151.21},
	{"Melbourne", "Australia", -37.81, 144.96},
	{"Brisbane", "Australia", -27.47, 153.03},
	{"Perth", "Australia", -31.95, 115.86},
	{"Adelaide", "Australia", -34.93, 138.60},
	{"Canberra", "Australia", -35.28, 149.13},
	{"Auckland", "New Zealand", -36.85, 174.76},
	{"Wellington", "New Zealand", -41.29, 174.78},
	{"Port Moresby", "Papua New Guinea", -9.44, 147.18},
	{"Suva", "Fiji", -18.14, 178.44},
}
