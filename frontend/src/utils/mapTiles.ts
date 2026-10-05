// Served by the local tiles container via nginx /tiles/ (see docker-compose.yml, frontend/nginx.conf).
export const MAP_TILE_URL = '/tiles/styles/basic-preview/{z}/{x}/{y}.png';

export const MAP_TILE_ATTRIBUTION =
    '&copy; <a href="https://www.openmaptiles.org/">OpenMapTiles</a> &copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>';

export const MAP_TILE_MAX_ZOOM = 19;
