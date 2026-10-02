import './App.css'
import Table from "./components/Table.tsx";
import Layout from "./layout/Layout.tsx";
import {QueryClient, QueryClientProvider} from "@tanstack/react-query";
import {Suspense, useState} from "react";
import Loading from "./components/Loading.tsx";
import {ErrorBoundary} from "react-error-boundary";
import Error from "./components/Error.tsx";
import SystemPicker from "./components/SystemPicker.tsx";

const queryClient = new QueryClient();

function systemFromHash(): string | undefined {
    const m = window.location.hash.match(/^#\/systems\/([^/]+)/);
    return m ? decodeURIComponent(m[1]) : undefined;
}

function App() {
    const [system, setSystem] = useState<string | undefined>(systemFromHash);

    const select = (name: string) => {
        window.location.hash = `#/systems/${encodeURIComponent(name)}`;
        setSystem(name);
    };

    return (
        <QueryClientProvider client={queryClient}>
            <Layout header={
                <ErrorBoundary fallback={<Error/>}>
                    <Suspense fallback={<Loading/>}>
                        <SystemPicker value={system} onChange={select}/>
                    </Suspense>
                </ErrorBoundary>
            }>
                <ErrorBoundary fallback={<Error/>}>
                    <Suspense fallback={<Loading/>}>
                        {system && <Table system={system}/>}
                    </Suspense>
                </ErrorBoundary>
            </Layout>
        </QueryClientProvider>
    )
}

export default App
