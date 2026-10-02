import useSystems from "../hooks/useSystems.ts";

interface SystemPickerProps {
    value?: string;
    onChange: (system: string) => void;
}

function SystemPicker({value, onChange}: SystemPickerProps) {
    const {systems} = useSystems();
    if (systems.length === 0) {
        return <span className="text-sm text-slate-500">No systems defined</span>;
    }
    const selected = (value && systems.some(s => s.name === value) ? value : systems[0].name) ?? "";
    if (selected && selected !== value) {
        onChange(selected);
    }
    if (systems.length === 1) {
        return <span className="text-sm text-slate-600">{systems[0].display_name ?? systems[0].name}</span>;
    }
    return <select className="rounded border border-slate-300 bg-white px-2 py-1 text-sm text-slate-700"
                   value={selected} onChange={e => onChange(e.target.value)}>
        {systems.map(s => <option key={s.name} value={s.name}>{s.display_name ?? s.name}</option>)}
    </select>
}

export default SystemPicker;
