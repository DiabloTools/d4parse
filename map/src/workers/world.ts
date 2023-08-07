import {MapData, Marker, WorldReq, WorldResp} from "./events";
import {Point} from "pixi.js";
import {
    defaultMarkerColor,
    getDisplayInfo,
    getWorldData, lookupSnoGroup,
    markerColors,
    markerMetaNames,
    Sno,
    snoGroupName, snoName
} from "./data";
import * as liqe from "liqe";

console.log("new world worker!");

self.onmessage = async (e: MessageEvent<WorldReq>) => {
    const data = await getWorldData(e.data.baseUrl, e.data.worldId);

    // Send map data
    if (e.data.retrieve.mapData) {
        self.postMessage({
            mapData: data as MapData,
        } as WorldResp);
    }

    // Add polygons
    if (e.data.retrieve.polygons) {
        for (let p of data.p ?? []) {
            const polygon = new Array<Point>();
            for (const wp of p) {
                polygon.push(new Point(wp[1], wp[0]));
            }

            self.postMessage({
                polygon: polygon,
            } as WorldResp);
        }
    }

    // Add markers
    if (e.data.retrieve.markers) {
        let query: liqe.LiqeQuery | undefined;
        if (e.data.query) {
            try {
                query = liqe.parse(e.data.query);
            } catch (e) {
                console.log("Error parsing search query:", e);
            }
        }

        for (let m of data.m ?? []) {
            const refGroup = await snoGroupName(m.g);

            if (query) {
                const refName = await snoName(m.g, m.r);
                const srcGroupId = await lookupSnoGroup(m.s);
                const srcGroup = await snoGroupName(srcGroupId);
                const srcName = await snoName(srcGroupId, m.s);

                const searchObj: any = {
                    id: String(m.r),
                    group: refGroup,
                    name: refName,
                    source_id: String(m.s),
                    source_group: srcGroup,
                    source: srcName,
                };

                if (!liqe.test(query, searchObj)) {
                    continue;
                }
            }

            const color = markerColors.get(refGroup) ?? defaultMarkerColor;

            // noinspection JSSuspiciousNameCombination
            self.postMessage({
                marker: {
                    color,
                    x: m.y, // Note: x and y are purposely swapped
                    y: m.x, // Note: x and y are purposely swapped
                    z: m.z,
                    w: 0.5, // TODO: configurable
                    h: 0.5,
                    ref: await getDisplayInfo( m.r, m.g),
                    source: await getDisplayInfo(m.s),
                    data: await Promise.all((m.d ?? []).map(
                        async (id: Sno.Id) => await getDisplayInfo(id),
                    )),
                    meta: Object.entries(m.m ?? {}).map(
                        ([k, v]) => [markerMetaNames.get(k) ?? k, v]
                    ),
                } as Marker,
            } as WorldResp);
        }
    }

    // Signal done
    self.postMessage({});
};

export {};