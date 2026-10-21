import { useEffect } from 'react';
import { useMap } from 'react-leaflet';
import {
    setWorkerUrl,
    type StyleSpecification,
    type VectorSourceSpecification,
} from 'maplibre-gl';
import { maplibreGL } from '@maplibre/maplibre-gl-leaflet';
import 'maplibre-gl/dist/maplibre-gl.css';
import workerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?url';
import { BASEMAP_STYLE } from '../utils/basemapStyle';
import { MAP_TILE_ATTRIBUTION } from '../utils/mapTiles';

// Same-origin worker. A blob: worker is blocked by the page CSP.
setWorkerUrl(workerUrl);

function styleForThisOrigin(): StyleSpecification {
    const style = structuredClone(BASEMAP_STYLE);
    const origin = window.location.origin;
    const source = style.sources.openmaptiles as VectorSourceSpecification;
    source.tiles = [`${origin}/tiles/data/v3/{z}/{x}/{y}.pbf`];
    source.attribution = MAP_TILE_ATTRIBUTION;
    style.glyphs = `${origin}/tiles/fonts/{fontstack}/{range}.pbf`;
    return style;
}

/** Vector basemap. Leaflet still owns markers, tracks, and popups. */
const VectorBasemap = () => {
    const map = useMap();

    useEffect(() => {
        const layer = maplibreGL({
            style: styleForThisOrigin(),
            fadeDuration: 0,
            refreshExpiredTiles: false,
        });
        layer.addTo(map);
        return () => {
            map.removeLayer(layer);
        };
    }, [map]);

    return null;
};

export default VectorBasemap;
