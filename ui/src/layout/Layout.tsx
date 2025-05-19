import React from "react";

interface LayoutProps {
    children: React.ReactNode | React.ReactNode[];
}

function Layout({children}: LayoutProps) {
    return <>
        <div className="bg-slate-100 h-20 flex items-center p-2 px-8 gap-4">
            <h3 className="text-2xl font-semibold text-slate-600">Strata</h3>
        </div>
        <div className="flex justify-center pt-8 container m-auto">
            {children}
        </div>
    </>
}

export default Layout;
