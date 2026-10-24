import { BrowserRouter, Routes, Route } from "react-router-dom";
import Layout from "./components/Layout";
import Home from "./pages/Home";
import Welcome from "./pages/Welcome";
import Help from "./pages/Help";
import Board from "./pages/Board";
import Projects from "./pages/Projects";
import Teams from "./pages/Teams";
import Tickets from "./pages/Tickets";

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<Home />} />
          <Route path="board" element={<Board />} />
          <Route path="welcome" element={<Welcome />} />
          <Route path="help" element={<Help />} />
          <Route path="projects" element={<Projects />} />
          <Route path="teams" element={<Teams />} />
          <Route path="tickets" element={<Tickets />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
