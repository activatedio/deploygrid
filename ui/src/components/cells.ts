import {components} from "../api/schema";

type Cell = components["schemas"]["DeploygridCell"];

export type ChipStatus = "success" | "warning" | "error" | "default" | "info";

export function cellStatus(cell: Cell): ChipStatus {
    if (cell.health === "Degraded") return "error";
    if (cell.drifted || cell.inconsistent) return "warning";
    if (cell.health === "Progressing") return "info";
    if (cell.health === "Healthy") return "success";
    return "default";
}

export function cellTitle(cell: Cell): string {
    const lines: string[] = [];
    lines.push(`Health: ${cell.health ?? "Unknown"}`);
    if (cell.cluster) lines.push(`Cluster: ${cell.cluster}${cell.namespace ? ` / ${cell.namespace}` : ""}`);
    if (cell.desired_version) lines.push(`Desired: ${cell.desired_version}`);
    if (cell.inconsistent) lines.push(`Inconsistent: more than one version is running`);
    for (const h of cell.hosts ?? []) lines.push(`Host: ${h}`);
    for (const a of cell.artifacts ?? []) {
        const versions = (a.versions ?? []).map(v => `${v.name}=${v.value}`).join(", ");
        lines.push(`${a.kind} ${a.namespace ? a.namespace + "/" : ""}${a.name}${versions ? `: ${versions}` : ""}`);
    }
    return lines.join("\n");
}
