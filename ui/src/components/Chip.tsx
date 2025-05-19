interface ChipProps {
    label: string;
    status: "success" | "warning" | "error" | "default";
}

function Chip({label, status}: ChipProps) {

    function getColor(): string {
        switch (status) {
            case "success":
                return 'bg-green-200';
            case "warning":
                return 'bg-amber-300';
            case "error":
                return 'bg-red-300';
            default:
                return 'bg-slate-200';
        }
    }

    return <div
        className={`w-20 flex select-none items-center ${getColor()} justify-center whitespace-nowrap rounded-lg font-sans font-bold uppercase text-slate-700 py-1.5 text-xs`}>
        <span>{label}</span>
    </div>
}

export default Chip;
