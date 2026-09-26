package place

// PlaceKind is the kind of a place in the geographic hierarchy.
//vogue:enum PlaceKind country,subdivision,city

// CountryCode is an ISO 3166-1 alpha-2 country code.
//vogue:string CountryCode trim upper required len=2 regex=^[A-Z]{2}$ example=AR

// PlaceName is the display name of a place.
//vogue:string PlaceName squish required min=1 max=200

// GeoNamesID identifies a GeoNames record.
//vogue:int GeoNamesID positive

// Population is the number of inhabitants of a place.
//vogue:int Population nonneg

// Elevation is the height of a place above sea level, in metres.
//vogue:int Elevation min=-500 max=9000 example=25

// Latitude is a WGS 84 latitude in decimal degrees.
//vogue:decimal Latitude min=-90 max=90 scale=6 example=-34.603722

// Longitude is a WGS 84 longitude in decimal degrees.
//vogue:decimal Longitude min=-180 max=180 scale=6 example=-58.381592

// PlaceID identifies a place across services.
//vogue:id PlaceID uuid7

// ImportRunID is the sequence number of a GeoNames import run.
//vogue:id ImportRunID int64
