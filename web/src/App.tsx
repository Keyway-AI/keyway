import { useEffect } from "react";
import { Route, Routes, useLocation } from "react-router-dom";
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
import Research from "./pages/Research";
import Login from "./pages/Login";
import Signup from "./pages/Signup";
import { CloudAuthProvider } from "./cloud/CloudAuth";
import CloudLayout from "./cloud/CloudLayout";
import CloudProjects from "./cloud/CloudProjects";
import CloudProject from "./cloud/CloudProject";
import NotFound from "./pages/NotFound";
import { Privacy, Terms } from "./pages/legal";

/** Per-route document titles — React 18 has no native <title> hoisting. */
const TITLES: Record<string, string> = {
  "/": "Keyway — Verify your token contracts",
  "/features": "Features — Keyway",
  "/pricing": "Pricing — Keyway",
  "/contact": "Contact — Keyway",
  "/research": "Research & methods — Keyway",
  "/privacy": "Privacy — Keyway",
  "/terms": "Terms — Keyway",
  "/login": "Sign in — Keyway",
  "/signup": "Get started — Keyway",
  "/cloud": "Keyway Cloud",
  "/app": "Dashboard — Keyway",
  "/app/findings": "Findings — Keyway",
  "/app/consumers": "Consumers — Keyway",
  "/app/changes": "Changes — Keyway",
  "/app/probes": "Probes — Keyway",
  "/app/blast-radius": "Blast radius — Keyway",
  "/app/canary": "Canary — Keyway",
  "/app/coverage": "Threat coverage — Keyway",
  "/app/agent": "Agent token inspector — Keyway",
};

function RouteTitle() {
  const { pathname } = useLocation();
  useEffect(() => {
    document.title =
      TITLES[pathname] ?? (pathname.startsWith("/cloud") ? "Keyway Cloud" : "Keyway");
  }, [pathname]);
  return null;
}

export default function App() {
  return (
    <ToastProvider>
      <RouteTitle />
      <Routes>
        <Route path="/" element={<Marketing />} />
        <Route path="/features" element={<Features />} />
        <Route path="/pricing" element={<Pricing />} />
        <Route path="/contact" element={<Contact />} />
        <Route path="/research" element={<Research />} />
        <Route path="/privacy" element={<Privacy />} />
        <Route path="/terms" element={<Terms />} />
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
        <Route
          path="/cloud"
          element={
            <CloudAuthProvider>
              <CloudLayout />
            </CloudAuthProvider>
          }
        >
          <Route index element={<CloudProjects />} />
          <Route path="projects/:id" element={<CloudProject />} />
        </Route>
        <Route path="*" element={<NotFound />} />
      </Routes>
    </ToastProvider>
  );
}
