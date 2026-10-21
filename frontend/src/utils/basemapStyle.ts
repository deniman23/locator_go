import type { StyleSpecification } from 'maplibre-gl';
import style from './basemapStyle.json';

// Same OpenMapTiles preview the tile server can paint as PNG, drawn in the browser instead.
export const BASEMAP_STYLE = style as unknown as StyleSpecification;
