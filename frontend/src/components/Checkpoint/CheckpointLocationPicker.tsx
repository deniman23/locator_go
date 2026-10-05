import React, { useMemo } from 'react';
import { MapContainer, TileLayer, Circle, Marker, useMap, useMapEvents } from 'react-leaflet';
import { MAP_TILE_ATTRIBUTION, MAP_TILE_MAX_ZOOM, MAP_TILE_URL } from '../../utils/mapTiles';
import L from 'leaflet';
import icon from 'leaflet/dist/images/marker-icon.png';
import iconShadow from 'leaflet/dist/images/marker-shadow.png';
import 'leaflet/dist/leaflet.css';

const DefaultIcon = L.icon({
    iconUrl: icon,
    shadowUrl: iconShadow,
    iconSize: [25, 41],
    iconAnchor: [12, 41],
});
L.Marker.prototype.options.icon = DefaultIcon;

const MINSK: [number, number] = [53.9, 27.57];

type ClickHandlerProps = {
    onPick: (lat: number, lng: number) => void;
};

const ClickHandler: React.FC<ClickHandlerProps> = ({ onPick }) => {
    useMapEvents({
        click(e) {
            onPick(e.latlng.lat, e.latlng.lng);
        },
    });
    return null;
};

const Recenter: React.FC<{ center: [number, number]; hasPoint: boolean }> = ({ center, hasPoint }) => {
    const map = useMap();
    const prev = React.useRef<string | null>(null);
    React.useEffect(() => {
        const key = `${center[0].toFixed(6)},${center[1].toFixed(6)}`;
        if (prev.current === key) return;
        const isFirst = prev.current == null;
        prev.current = key;
        if (!hasPoint) {
            map.setView(center, 11);
            return;
        }
        if (isFirst) {
            map.setView(center, 15);
        } else {
            map.panTo(center);
        }
    }, [map, center, hasPoint]);
    return null;
};

type Props = {
    latitude: string;
    longitude: string;
    radius: string;
    onChange: (lat: number, lng: number) => void;
};

const CheckpointLocationPicker: React.FC<Props> = ({ latitude, longitude, radius, onChange }) => {
    const lat = parseFloat(latitude);
    const lng = parseFloat(longitude);
    const rad = parseFloat(radius);
    const hasPoint = Number.isFinite(lat) && Number.isFinite(lng);
    const center = useMemo<[number, number]>(
        () => (hasPoint ? [lat, lng] : MINSK),
        [hasPoint, lat, lng],
    );

    return (
        <div className="checkpoint-location-picker">
            <p className="filter-hint">Кликните по карте, чтобы указать координаты. Радиус — в поле ниже.</p>
            <div className="checkpoint-picker-map">
                <MapContainer
                    center={MINSK}
                    zoom={11}
                    style={{ height: '100%', width: '100%' }}
                    scrollWheelZoom
                >
                    <TileLayer
                        url={MAP_TILE_URL}
                        attribution={MAP_TILE_ATTRIBUTION}
                        maxZoom={MAP_TILE_MAX_ZOOM}
                    />
                    <Recenter center={center} hasPoint={hasPoint} />
                    <ClickHandler onPick={onChange} />
                    {hasPoint && (
                        <>
                            <Marker
                                position={[lat, lng]}
                                draggable
                                eventHandlers={{
                                    dragend: (e) => {
                                        const m = e.target as L.Marker;
                                        const p = m.getLatLng();
                                        onChange(p.lat, p.lng);
                                    },
                                }}
                            />
                            {Number.isFinite(rad) && rad > 0 && (
                                <Circle
                                    center={[lat, lng]}
                                    radius={rad}
                                    pathOptions={{ color: '#2563eb', fillOpacity: 0.15 }}
                                />
                            )}
                        </>
                    )}
                </MapContainer>
            </div>
        </div>
    );
};

export default CheckpointLocationPicker;
