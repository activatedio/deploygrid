interface ChipProps {
    label: string;
    status: "success" | "warning" | "error" | "default" | "info";
    sub?: string;
    title?: string;
}

function Chip({label, status, sub, title}: ChipProps) {

    function getColor(): string {
        switch (status) {
            case "success":
                return 'bg-green-200';
            case "warning":
                return 'bg-amber-300';
            case "error":
                return 'bg-red-300';
            case "info":
                return 'bg-sky-200';
            default:
                return 'bg-slate-200';
        }
    }

    return <div title={title}
                className={`min-w-20 inline-flex flex-col select-none items-center ${getColor()} justify-center whitespace-nowrap rounded-lg font-sans font-bold text-slate-700 py-1.5 px-2 text-xs`}>
        <span>{label}</span>
        {sub && <span className="font-normal text-slate-600">{sub}</span>}
    </div>
}

export default Chip;
