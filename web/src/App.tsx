import { Route, Routes } from "react-router-dom";
import Layout from "./components/Layout";
import Dashboard from "./pages/Dashboard";
import Consumers from "./pages/Consumers";
import Changes from "./pages/Changes";
import BlastRadius from "./pages/BlastRadius";
import Canary from "./pages/Canary";

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Dashboard />} />
        <Route path="consumers" element={<Consumers />} />
        <Route path="changes" element={<Changes />} />
        <Route path="blast-radius" element={<BlastRadius />} />
        <Route path="canary" element={<Canary />} />
      </Route>
    </Routes>
  );
}
