import { Route, Routes } from "react-router-dom";
import Layout from "./components/Layout";
import { ToastProvider } from "./components/toast";
import Dashboard from "./pages/Dashboard";
import Findings from "./pages/Findings";
import Consumers from "./pages/Consumers";
import Changes from "./pages/Changes";
import BlastRadius from "./pages/BlastRadius";
import Canary from "./pages/Canary";
import Probes from "./pages/Probes";

export default function App() {
  return (
    <ToastProvider>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<Dashboard />} />
          <Route path="findings" element={<Findings />} />
          <Route path="consumers" element={<Consumers />} />
          <Route path="changes" element={<Changes />} />
          <Route path="probes" element={<Probes />} />
          <Route path="blast-radius" element={<BlastRadius />} />
          <Route path="canary" element={<Canary />} />
        </Route>
      </Routes>
    </ToastProvider>
  );
}
