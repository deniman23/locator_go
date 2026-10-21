// The browser draws the map from local vector tiles (/tiles/data/v3/…).
// Painting PNG on the server stalled for tens of seconds when many tiles
// were requested together.

export const MAP_TILE_ATTRIBUTION =
    '&copy; <a href="https://www.openmaptiles.org/">OpenMapTiles</a> &copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>';

// Vector data in belarus.mbtiles stops at z14. MapLibre overzooms past that.
export const MAP_TILE_MAX_ZOOM = 18;
