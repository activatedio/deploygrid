import './App.css'
import Table from "./components/Table.tsx";
import Layout from "./layout/Layout.tsx";
import {QueryClient, QueryClientProvider} from "@tanstack/react-query";
import {Suspense, useState} from "react";
import Loading from "./components/Loading.tsx";
import {ErrorBoundary} from "react-error-boundary";
import Error from "./components/Error.tsx";
import SystemPicker from "./components/SystemPicker.tsx";
import ComponentDetail from "./components/ComponentDetail.tsx";

const queryClient = new QueryClient();

interface Route {
    system?: string;
    component?: string;
}

function routeFromHash(): Route {
    const m = window.location.hash.match(/^#\/systems\/([^/]+)(?:\/components\/([^/]+))?/);
    if (!m) return {};
    return {system: decodeURIComponent(m[1]), component: m[2] ? decodeURIComponent(m[2]) : undefined};
}

function App() {
    const [route, setRoute] = useState<Route>(routeFromHash);
    const system = route.system;

    const navigate = (r: Route) => {
        window.location.hash = r.system
            ? `#/systems/${encodeURIComponent(r.system)}${r.component ? `/components/${encodeURIComponent(r.component)}` : ""}`
            : "";
        setRoute(r);
    };

    return (
        <QueryClientProvider client={queryClient}>
            <Layout header={
                <ErrorBoundary fallback={<Error/>}>
                    <Suspense fallback={<Loading/>}>
                        <SystemPicker value={system} onChange={name => navigate({system: name})}/>
                    </Suspense>
                </ErrorBoundary>
            }>
                <ErrorBoundary fallback={<Error/>}>
                    <Suspense fallback={<Loading/>}>
                        {system && route.component
                            ? <ComponentDetail system={system} component={route.component} onBack={() => navigate({system})}/>
                            : system && <Table system={system} onSelect={component => navigate({system, component})}/>}
                    </Suspense>
                </ErrorBoundary>
            </Layout>
        </QueryClientProvider>
    )
}

export default App
