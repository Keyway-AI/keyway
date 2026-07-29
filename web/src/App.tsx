import { Navigate, Route, Routes } from "react-router-dom";
import Layout from "./components/Layout";
import { ToastProvider } from "./components/toast";
import Dashboard from "./pages/Dashboard";
import Findings from "./pages/Findings";
import Consumers from "./pages/Consumers";
import Changes from "./pages/Changes";
import BlastRadius from "./pages/BlastRadius";
import Canary from "./pages/Canary";
import Probes from "./pages/Probes";
import Coverage from "./pages/Coverage";
import Agent from "./pages/Agent";
import Marketing from "./pages/Marketing";
import Features from "./pages/Features";
import Pricing from "./pages/Pricing";
import Contact from "./pages/Contact";
import Login from "./pages/Login";
import Signup from "./pages/Signup";

export default function App() {
  return (
    <ToastProvider>
      <Routes>
        <Route path="/" element={<Marketing />} />
        <Route path="/features" element={<Features />} />
        <Route path="/pricing" element={<Pricing />} />
        <Route path="/contact" element={<Contact />} />
        <Route path="/login" element={<Login />} />
        <Route path="/signup" element={<Signup />} />
        <Route path="/app" element={<Layout />}>
          <Route index element={<Dashboard />} />
          <Route path="findings" element={<Findings />} />
          <Route path="consumers" element={<Consumers />} />
          <Route path="changes" element={<Changes />} />
          <Route path="probes" element={<Probes />} />
          <Route path="blast-radius" element={<BlastRadius />} />
          <Route path="canary" element={<Canary />} />
          <Route path="coverage" element={<Coverage />} />
          <Route path="agent" element={<Agent />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </ToastProvider>
  );
}
