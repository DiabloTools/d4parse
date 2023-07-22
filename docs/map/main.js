function pxToMapUnit(px) {
    return [px[0] / pxPerMapUnit, px[1] / pxPerMapUnit];
}

function subPx(a, b) {
    return [a[0] - b[0], a[1] - b[1]];
}

// Markers
const markerColors = {
    'Actor': 'green',
    'AmbientSound': 'lightpurple',
    'Encounter': 'red',
    'EffectGroup': 'blue',
    'FogVolume': 'mintcream',
    'Light': 'yellow',
    'MarkerSet': 'lightgray',
    'Material': 'orange',
    'Particle': 'brown',
    'Quest': 'lightblue',
    'Sound': 'purple',
    'Unknown': 'darkgray',
    'Weather': 'lightskyblue',
};

const worldSnoGroup = 48;
const sceneSnoGroup = 33;
const defaultSnoId =  69068;
const defaultSnoName = "Sanctuary_Eastern_Continent"

// Load groups and names data
loadData((groups, names) => {
    $(() => {
        // Add world selector
        const worldSelect = $("#worldSelect");
        worldSelect.select2({
            theme: "classic"
        });

        let worldSnos = Object.entries(names[worldSnoGroup]);
        worldSnos.sort((a,b) => a[1].localeCompare(b[1]))
        for (const [snoId, snoName] of worldSnos) {
            worldSelect.append(`<option value="${snoId}">[World] ${snoName}</option>`);
        }

        let sceneSnos = Object.entries(names[sceneSnoGroup]);
        sceneSnos.sort((a,b) => a[1].localeCompare(b[1]))
        for (const [snoId, snoName] of sceneSnos) {
            worldSelect.append(`<option value="${snoId}">[Scene] ${snoName}</option>`);
        }

        // Load base world
        loadWorld(groups, names, defaultSnoId, defaultSnoName);

        worldSelect.change(function() {
            loadWorld(groups, names, $(this).val(), $(this).find('option:selected').text());
        });

        // Remove loading screen
        $("#loading").hide();
    });
});

// TODO: add overlays for quest conditioned map updates
// TODO: add radius (on hover) around markers with a radius
// TODO: add rotated and non-rotated grid
// TODO: filter by gizmo type
// TODO: expand marker sets on click
// TODO: custom search with Fuse?

function rotate(p, angle) {
    const rads = (Math.PI / 180) * angle;
    const cos = Math.cos(rads);
    const sin = Math.sin(rads);
    return L.point(
        (cos * p.x) + (sin * p.y),
        (cos * p.y) - (sin * p.x)
    );
}

function markerPopup(marker, title) {
    let extra = '';
    const meta = marker.m ?? {};
    if (meta.hasOwnProperty('mt')) {
        extra += `Marker Type: ${meta.mt}<br/>`
    }
    if (meta.hasOwnProperty('gt')) {
        extra += `Gizmo Type: ${meta.gt}<br/>`
    }

    return `<b><a href="../sno/${marker.r}.html">${title}</a></b>
    <p>
    Source: <a href="../sno/${marker.s}.html">${marker.s}</a>
    <br/>
    ${extra}
    <br/>
    <i>${marker.x}, ${marker.y}, ${marker.z}</i>
    </p>`;
}

function snoGroupName(groups, id) {
    if (id === 255) {
        return "Unknown";
    }
    return groups[id] ?? `Group_${id}`;
}

function snoName(groups, names, group, id) {
    if (group > 250 || !names.hasOwnProperty(group)) {
        return `[Unknown] ${id === -1 ? 'Unknown' : id}`;
    }

    const groupName = snoGroupName(groups, group);
    names = names[group];

    if (!names.hasOwnProperty(id)) {
        return `[${groupName}] ${id}`
    }

    return `[${groupName}] ${names[id]}`
}

function loadWorld(groups, names, worldSnoId, worldSnoName) {
    binaryRequest('GET', `data/${worldSnoId}.mpk`).then((data) => {
        const mapData = msgpackr.unpack(data);

        if (!mapData.p && !mapData.m) {
            alert("No Data For Scene/World");
            return
        }

        if (window.m && window.m.remove) {
            window.m.remove();
        }

        $("#worldSelect").val(worldSnoId);

        // From ZoneMapParams in WorldDefinition
        window.zoneArtScale = mapData.artScale; // tZoneMapParams.fZoneArtScale
        window.zoneArtCenter = [mapData.artCenterX, mapData.artCenterY]; // tZoneMapParams.vecZoneArtCenter
        window.zoneMapParamsScale = 5; // Scale of texture relative to zone map params

        // Pixels <-> Leaflet map units
        window.mapUnitPerTile = 64;
        window.mapSize = 40;
        window.tileSize = 512;
        window.pxPerMapUnit = tileSize / mapUnitPerTile;

        // Calculated constants
        window.min = [0, 0];
        window.max = [tileSize * mapSize, tileSize * mapSize];
        window.origin = [zoneArtCenter[0] * zoneMapParamsScale, zoneArtCenter[1] * zoneMapParamsScale];
        window.ptScale = 1 + ((1 - zoneArtScale) * zoneMapParamsScale);

        window.originMapUnits = pxToMapUnit(origin);
        window.minMapUnits = subPx(pxToMapUnit(min), originMapUnits);
        window.maxMapUnits = subPx(pxToMapUnit(max), originMapUnits);

        // D4 CRS (TODO: determine from world data)
        const D4Projection = L.extend({}, L.Projection.LonLat, {
            project: function (latlng) {
                let point = L.Projection.LonLat.project(latlng);
                return rotate(point, -45);
            },
            unproject: function (point) {
                point = rotate(point, 45);
                return L.Projection.LonLat.unproject(point);
            },
        });

        const D4CRS = L.extend({}, L.CRS.Simple, {
            projection: D4Projection,
            transformation: new L.Transformation(ptScale, originMapUnits[0], ptScale, originMapUnits[1]),
        });

        // Setup renderer
        const canvas = L.canvas();

        // Setup map
        window.m = L.map('map', {
            attributionControl: false,
            crs: D4CRS,
            renderer: canvas,
            // maxBounds: L.latLngBounds( // Basically magic at this point
            //     L.latLng(-970, -2890),
            //     L.latLng(-970, 2545),
            // )
        }).setView([0, 0], 0);

        worldTileLayer(window.m, worldSnoId, worldSnoName);

        // Add map events
        window.m.on('click', function (e) {
            L.popup()
                .setLatLng(e.latlng)
                .setContent(`${e.latlng.lat}, ${e.latlng.lng}`)
                .openOn(window.m);
        });

        // Add markers
        L.circleMarker([0, 0], {
            radius: 5,
            stroke: false,
            fill: true,
            fillOpacity: 0.75,
            fillColor: "black",
        }).bindTooltip("This is the center of the world!").addTo(window.m);

        // Load world
        const p = mapData.p ?? [];
        const m = mapData.m ?? [];

        // Polygons
        let len = p.length;
        while (len--) { // Using while has a measurable performance improvement... bc Javascript.
            L.polygon(p[len], {
                weight: 3,
                color: '#ffffff',
                fill: false,
                opacity: 0.1,
                interactive: false,
            }).addTo(window.m)
        }

        // Markers
        const markers = {};

        len = m.length;
        while (len--) {
            const marker = m[len];
            const groupName = snoGroupName(groups, marker.g);
            const title = snoName(groups, names, marker.g, marker.r);

            const circle = L.circleMarker([marker.x, marker.y], {
                radius: 5,
                stroke: false,
                fill: true,
                fillOpacity: 0.75,
                fillColor: markerColors[groupName],
            }).bindPopup(
                markerPopup(marker, title),
                {direction: 'center'},
            );

            if (!markers.hasOwnProperty(groupName)) {
                markers[groupName] = L.markerClusterGroup({
                    spiderfyOnMaxZoom: false,
                    removeOutsideVisibleBounds: true,
                    disableClusteringAtZoom: 3,
                    iconCreateFunction: function (cluster) {
                        var childCount = cluster.getChildCount();
                        return new L.DivIcon({
                            html: `<div>${childCount}</div>`,
                            className: `cluster group_${groupName}`,
                            iconSize: new L.Point(40, 40),
                        });
                    }
                });
            }
            markers[groupName].addLayer(circle);
        }

        const dataLayers = L.control.layers({}, {}).addTo(window.m);
        for (const markerGroup of Object.keys(markers).sort()) {
            const layer = markers[markerGroup];
            dataLayers.addOverlay(layer, markerGroup);
        }
    }, console.error);
}

function worldTileLayer(map, worldSnoId, worldSnoName) {
    // Setup tiles
    return L.tileLayer(`maptiles/${worldSnoId}/{z}/{x}_{y}.png`, {
        tileSize: tileSize,
        maxZoom: 15,
        minZoom: -1,
        minNativeZoom: 0,
        maxNativeZoom: 3,
        noWrap: true,
        tms: false,
    }).addTo(map);
}

function loadData(cb) {
    Promise.all([
        binaryRequest('GET', '../groups.mpk'),
        binaryRequest('GET', '../names.mpk'),
    ]).then((values) => {
        cb(
            msgpackr.unpack(values[0]),
            msgpackr.unpack(values[1]),
        );
    }, console.error);
}

function binaryRequest(method, url) {
    var xhr = new XMLHttpRequest();
    xhr.responseType = 'arraybuffer';
    return $.ajax({
        method,
        url,
        xhr: function() {
            return xhr;
        }
    })
    //
    // return new Promise(function (resolve, reject) {
    //     const xhr = new XMLHttpRequest();
    //     xhr.open(method, url);
    //     xhr.responseType = 'arraybuffer';
    //     xhr.onload = resolve;
    //     xhr.onerror = reject;
    //     xhr.send();
    // });
}
