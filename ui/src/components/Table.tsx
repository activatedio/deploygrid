import useGrid from "../hooks/useGrid.ts";
import {components} from "../api/schema";
import {Fragment, ReactNode, useState} from "react";
import Chip from "./Chip.tsx";
import {cellStatus, cellTitle} from "./cells.ts";

type Cell = components["schemas"]["DeploygridCell"];
type Row = components["schemas"]["DeploygridGridRow"];
type Environment = components["schemas"]["DeploygridEnvironment"];

interface TableProps {
    system: string;
    onSelect: (component: string) => void;
}

const thClass = `px-6 py-3 text-start text-xs font-medium text-slate-600`;
const tdClass = `px-6 py-3 whitespace-nowrap text-sm font-medium text-gray-800 align-top`;

function rowNeedsAttention(row: Row): boolean {
    return Object.values(row.cells ?? {}).some(c => c.drifted || c.inconsistent || c.health === "Degraded")
        || (row.children ?? []).some(rowNeedsAttention);
}

function CellView({cell}: { cell?: Cell }) {
    if (!cell) {
        return <span className="text-slate-300">--</span>;
    }
    const label = cell.version || (cell.desired_version ? "?" : "--");
    const sub = cell.drifted && cell.desired_version ? `wants ${cell.desired_version}` : undefined;
    const host = cell.hosts?.[0];
    return <div className="flex flex-col gap-1">
        <Chip status={cellStatus(cell)} label={label} sub={sub} title={cellTitle(cell)}/>
        {host && <a href={`https://${host}`} target="_blank" rel="noreferrer"
                    className="text-xs text-sky-600 hover:underline truncate max-w-40" title={host}>{host}</a>}
    </div>;
}

function RowHeader({row, indent, onSelect}: { row: Row; indent: number; onSelect: (component: string) => void }) {
    const c = row.component ?? {name: "?"};
    const firstCell = Object.values(row.cells ?? {})[0];
    return <div className="flex flex-col gap-1" style={{paddingLeft: `${indent}em`}}>
        <span>
            <button className="hover:underline text-left" onClick={() => c.name && onSelect(c.name)}>{c.display_name ?? c.name}</button>
            {c.discovered && <span className="ml-2 text-xs font-normal text-slate-400" title="No Component resource declares this row">discovered</span>}
        </span>
        <span className="text-xs text-gray-400">
            {c.kind}
            {firstCell?.links?.map(l => (
                <a key={l.name} href={l.url} target="_blank" rel="noreferrer" className="ml-2 text-sky-600 hover:underline">{l.name}</a>
            ))}
        </span>
    </div>
}

function renderRows(rows: Row[], environments: Environment[], indent: number, onSelect: (component: string) => void, attentionOnly: boolean): ReactNode[] {
    const out: ReactNode[] = [];
    for (const row of rows) {
        if (attentionOnly && !rowNeedsAttention(row)) continue;
        const name = row.component?.name ?? "";
        out.push(<tr key={name}>
            <td className={tdClass}><RowHeader row={row} indent={indent} onSelect={onSelect}/></td>
            {environments.map(env => (
                <td className={tdClass} key={env.name}>
                    <CellView cell={env.name ? row.cells?.[env.name] : undefined}/>
                </td>
            ))}
        </tr>);
        if (row.children?.length) {
            out.push(...renderRows(row.children, environments, indent + 1, onSelect, attentionOnly));
        }
    }
    return out;
}

function Table({system, onSelect}: TableProps) {
    const {data} = useGrid(system);
    const [attentionOnly, setAttentionOnly] = useState(false);
    const environments = data.environments ?? [];
    const groups = (data.groups ?? []).filter(g => !attentionOnly || (g.rows ?? []).some(rowNeedsAttention));
    const notices = [...(data.errors ?? []), ...(data.warnings ?? [])];

    return <div className="flex flex-col gap-4">
        <label className="flex items-center gap-2 text-sm text-slate-600 self-end">
            <input type="checkbox" checked={attentionOnly} onChange={e => setAttentionOnly(e.target.checked)}/>
            Only drifted, inconsistent or degraded
        </label>
        {notices.length > 0 && (
            <ul className="rounded-lg border border-amber-300 bg-amber-50 px-4 py-2 text-xs text-amber-900">
                {notices.map((n, i) => <li key={i}>{n}</li>)}
            </ul>
        )}
        <div className="inline-block border rounded-lg overflow-hidden border-slate-300">
            <table className="divide-y divide-slate-200 text-left font-light">
                <thead className="bg-slate-50">
                <tr>
                    <th className={`${thClass} border-r-2 border-slate-100`}>Component</th>
                    {environments.map(e => <th key={e.name} className={thClass}>{e.display_name ?? e.name}</th>)}
                </tr>
                </thead>
                <tbody className="divide-y divide-slate-200">
                {groups.length === 0 && (
                    <tr>
                        <td className={tdClass} colSpan={environments.length + 1}>
                            <span className="text-slate-400">{attentionOnly ? "Nothing needs attention." : "Nothing observed yet for this system."}</span>
                        </td>
                    </tr>
                )}
                {groups.map(group => (
                    <Fragment key={group.name}>
                        <tr className="bg-slate-50">
                            <td className={`${tdClass} font-bold`} colSpan={environments.length + 1}>{group.display_name ?? group.name}</td>
                        </tr>
                        {renderRows(group.rows ?? [], environments, 0, onSelect, attentionOnly)}
                    </Fragment>
                ))}
                </tbody>
            </table>
        </div>
    </div>
}

export default Table;
