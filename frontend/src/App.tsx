import { BrowserRouter, Routes, Route, Outlet } from "react-router-dom";
import { Suspense } from "react";
import { Provider } from "jotai";

import "./App.css";
import { Header } from "./components/Header";
import { Home } from "./pages/Home";
import { FlowList } from "./components/FlowList";
import { FlowView } from "./pages/FlowView";

export const TEXT_FILTER_KEY = "text";
export const SERVICE_FILTER_KEY = "service";
export const START_FILTER_KEY = "start";
export const END_FILTER_KEY = "end";

function App() {
  return (
    <Provider>
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
          </Route>
        </Routes>
      </BrowserRouter>
    </Provider>
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
