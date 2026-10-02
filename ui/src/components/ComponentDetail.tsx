import useComponent from "../hooks/useComponent.ts";
import useHistory from "../hooks/useHistory.ts";
import useSystem from "../hooks/useSystem.ts";
import {components} from "../api/schema";
import Chip from "./Chip.tsx";
import {cellStatus} from "./cells.ts";

type Cell = components["schemas"]["DeploygridCell"];
type Change = components["schemas"]["HistoryChange"];

interface ComponentDetailProps {
    system: string;
    component: string;
    onBack: () => void;
}

function changeText(c: Change): string {
    if (!c.from && c.to) return `${c.to} deployed`;
    if (c.from && !c.to) return `${c.from} no longer observed`;
    return `${c.from} → ${c.to}`;
}

function CellCard({env, cell}: { env: string; cell?: Cell }) {
    return <div className="rounded-lg border border-slate-200 p-4 flex flex-col gap-2 min-w-56">
        <div className="flex items-center justify-between">
            <span className="text-sm font-semibold text-slate-700">{env}</span>
            {cell ? <Chip status={cellStatus(cell)} label={cell.version || "--"}/> :
                <span className="text-slate-300 text-sm">not observed</span>}
        </div>
        {cell && <dl className="text-xs text-slate-600 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1">
            {cell.desired_version && <><dt className="text-slate-400">desired</dt><dd>{cell.desired_version}{cell.drifted && <span className="ml-1 text-amber-600">(drift)</span>}</dd></>}
            <dt className="text-slate-400">health</dt><dd>{cell.health}</dd>
            {(cell.clusters?.length ?? 0) > 1
                ? <><dt className="text-slate-400">clusters</dt><dd>{cell.clusters!.join(", ")}</dd></>
                : cell.cluster && <><dt className="text-slate-400">cluster</dt><dd>{cell.cluster}{cell.namespace ? ` / ${cell.namespace}` : ""}</dd></>}
            {cell.inconsistent && <><dt className="text-slate-400">note</dt><dd className="text-amber-600">more than one version running</dd></>}
        </dl>}
        {cell?.hosts?.length ? <ul className="text-xs">
            {cell.hosts.map(h => <li key={h}><a className="text-sky-600 hover:underline" href={`https://${h}`} target="_blank" rel="noreferrer">{h}</a></li>)}
        </ul> : null}
        {cell?.links?.length ? <div className="text-xs flex gap-2 flex-wrap">
            {cell.links.map(l => <a key={l.name} className="text-sky-600 hover:underline" href={l.url} target="_blank" rel="noreferrer">{l.name}</a>)}
        </div> : null}
        {cell?.artifacts?.length ? <details className="text-xs text-slate-600">
            <summary className="cursor-pointer text-slate-500">{cell.artifacts.length} artifact{cell.artifacts.length === 1 ? "" : "s"}</summary>
            <ul className="mt-1 flex flex-col gap-1">
                {cell.artifacts.map((a, i) => <li key={i}>
                    <span className="font-mono">{a.kind}</span> {a.namespace ? `${a.namespace}/` : ""}{a.name}
                    {a.versions?.length ? <span className="text-slate-400"> · {a.versions.map(v => `${v.name}=${v.value}`).join(", ")}</span> : null}
                </li>)}
            </ul>
        </details> : null}
    </div>
}

function ComponentDetail({system, component, onBack}: ComponentDetailProps) {
    const {system: sys} = useSystem(system);
    const {row} = useComponent(system, component);
    const {changes} = useHistory(system, component);
    const environments = sys.environments ?? [];
    const c = row.component ?? {name: component};

    return <div className="flex flex-col gap-6 w-full max-w-5xl">
        <div className="flex items-center gap-4">
            <button onClick={onBack} className="text-sm text-sky-600 hover:underline">← grid</button>
            <h2 className="text-xl font-semibold text-slate-700">{c.display_name ?? c.name}</h2>
            <span className="text-xs text-slate-400">{c.kind}{c.discovered ? " · discovered" : ""}</span>
        </div>
        {c.description && <p className="text-sm text-slate-600">{c.description}</p>}
        <div className="flex gap-4 flex-wrap">
            {environments.map(e => <CellCard key={e.name} env={e.display_name ?? e.name ?? ""} cell={e.name ? row.cells?.[e.name] : undefined}/>)}
        </div>
        <div>
            <h3 className="text-sm font-semibold text-slate-700 mb-2">Recent changes</h3>
            {changes.length === 0 ? <p className="text-sm text-slate-400">No version changes observed since the server started. Durable history is recorded as Kubernetes Events on the Component.</p> :
                <table className="text-sm text-left">
                    <thead className="text-xs text-slate-500">
                    <tr><th className="pr-6 py-1">When</th><th className="pr-6 py-1">Environment</th><th className="pr-6 py-1">Change</th><th className="py-1">Where</th></tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                    {changes.map((ch, i) => <tr key={i}>
                        <td className="pr-6 py-1 whitespace-nowrap text-slate-500">{ch.observed_at ? new Date(ch.observed_at).toLocaleString() : ""}</td>
                        <td className="pr-6 py-1">{ch.environment}</td>
                        <td className="pr-6 py-1 font-mono">{changeText(ch)}</td>
                        <td className="py-1 text-slate-500">{ch.cluster}{ch.namespace ? ` / ${ch.namespace}` : ""}</td>
                    </tr>)}
                    </tbody>
                </table>}
        </div>
    </div>
}

export default ComponentDetail;
