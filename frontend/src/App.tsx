import { BrowserRouter, Routes, Route, Outlet } from "react-router-dom";
import { Suspense } from "react";

import "./App.css";
import { Header } from "./components/Header";
import { Home } from "./pages/Home";
import { FlowList } from "./components/FlowList";
import { FlowView } from "./pages/FlowView";
import { DiffView } from "./pages/DiffView";
import { Corrie } from "./components/Corrie";

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Home />} />
          <Route
            path="flow/:id"
            element={
              <Suspense>
                <FlowView />
              </Suspense>
            }
          />
          <Route
            path="diff/:id"
            element={
              <Suspense>
                <DiffView />
              </Suspense>
            }
          />
          <Route
            path="corrie/"
            element={
              <Suspense>
                <Corrie />
              </Suspense>
            }
          />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

function Layout() {
  return (
    <div className="grid-container">
      <header className="header-area">
        <div className="header">
          <Header></Header>
        </div>
      </header>
      <aside className="flow-list-area">
        <Suspense>
          <FlowList></FlowList>
        </Suspense>
      </aside>
      <main className="flow-details-area">
        <Outlet />
      </main>
      <footer className="footer-area"></footer>
    </div>
  );
}
export default App;
