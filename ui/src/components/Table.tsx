import useGrid from "../hooks/useGrid.ts";
import {components} from "../api/schema";
import {Fragment, ReactNode, useCallback} from "react";
import Chip from "./Chip.tsx";


function Table() {
    const {data} = useGrid();
    const {environments, components} = data;

    const thClass = `px-6 py-3 text-start text-xs font-medium text-slate-600`;
    const tdClass = `px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-800`;

    const rowHeader = (name: string, type: string): ReactNode => {
        return <div className="flex flex-col gap-1">
            <span>{name}</span>
            <span className="text-gray-400">
                {type}
            </span>
        </div>
    }

    const hasDeployments = (r: components["schemas"]["DeploygridComponent"]): boolean => {
        return Boolean(r.deployments)
    }

    const indentRenderer = useCallback((child: ReactNode, indent: number, important?: boolean) => {
        return <td className={tdClass}
                   style={{textIndent: `${indent}em`, fontWeight: important ? "bold" : "normal"}}>{child}</td>
    }, [tdClass]);

    const getDeploymentVersionForEnv = (r: components["schemas"]["DeploygridComponent"], env: string | undefined): string => {
        const noVersion = '--';
        if (!env) {
            return noVersion;
        }
        if (r.deployments) {
            if (env in r.deployments) {
                return r.deployments[env].version ?? noVersion;
            }
        }
        return noVersion;
    }

    //todo get mapping of deployment key to td index in table
    const rowRenderer = useCallback((r: components["schemas"]["DeploygridComponent"], indent: number): ReactNode => {
        if (r.component_type === "Group") {
            return <tr key={r.name}>
                {indentRenderer(<span>{r.name}</span>, indent, true)}
            </tr>
        } else {
            if (hasDeployments(r)) {
                return <tr key={r.name}>
                    {indentRenderer(rowHeader(r.name ?? '', r.component_type ?? ''), indent)}
                    {environments?.map((env) => (
                        <td className={tdClass} key={env?.name ?? ''}>
                            <Chip status="success" label={getDeploymentVersionForEnv(r, env.name)}/>
                        </td>
                    ))}
                </tr>
            }
        }
        return <td></td>
    }, [indentRenderer, tdClass]);

    const transformTree = useCallback((node: components["schemas"]["DeploygridComponent"], indent: number): ReactNode[] => {

        if (node == null) return [];
        const rows: ReactNode[] = [];

        const row = rowRenderer(node, indent);
        rows.push(row);
        node.children?.forEach(child => {
            rows.push(transformTree(child, indent + 1));
        })

        return rows;

    }, [rowRenderer])

    const populatedTree = useCallback(() => {
        return components?.map((component) => (
            <Fragment key={component.name}>
                {transformTree(component, 0) ?? <tr></tr>}
            </Fragment>
        ))
    }, [components, transformTree])

    return <div className="inline-block border rounded-lg overflow-hidden border-slate-300 overflow-x-visible">
        <table className="divide-y divide-slate-200 text-left font-light">
            <thead className="bg-slate-50">
            <tr>
                <th className={`${thClass} border-r-2 border-slate-100`}>Component</th>
                {environments?.map(((e) => <th key={e.name} className={`capitalize ${thClass}`}>{e.name}</th>))}
            </tr>
            </thead>
            <tbody className="divide-y divide-slate-200">
            {populatedTree()}
            </tbody>
        </table>
    </div>
}

export default Table;
