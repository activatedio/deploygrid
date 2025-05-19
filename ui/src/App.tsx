import './App.css'
import Table from "./components/Table.tsx";
import Layout from "./layout/Layout.tsx";
import {QueryClient, QueryClientProvider} from "@tanstack/react-query";
import {Suspense} from "react";
import Loading from "./components/Loading.tsx";
import {ErrorBoundary} from "react-error-boundary";
import Error from "./components/Error.tsx";

const queryClient = new QueryClient();

function App() {

    return (
        <QueryClientProvider client={queryClient}>
            <Layout>
                <ErrorBoundary fallback={<Error/>}>
                    <Suspense fallback={<Loading/>}>
                        <Table/>
                    </Suspense>
                </ErrorBoundary>
            </Layout>
        </QueryClientProvider>
    )
}

export default App
