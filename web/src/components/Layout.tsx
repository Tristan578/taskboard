import { useState } from "react";
import { NavLink, Outlet } from "react-router-dom";
import {
  LayoutDashboard,
  FolderKanban,
  Users,
  ListTodo,
  Sparkles,
  TerminalSquare,
  CircleHelp,
  Plus,
  Menu,
  X,
} from "lucide-react";
import TerminalPanel from "./TerminalPanel";
const items = [
  { to: "/board", icon: LayoutDashboard, label: "My board" },
  { to: "/welcome", icon: Plus, label: "New board" },
  { to: "/projects", icon: FolderKanban, label: "Boards" },
  { to: "/help", icon: CircleHelp, label: "Help & connections" },
];
export default function Layout() {
  const [terminalOpen, setTerminalOpen] = useState(false);
  const [menu, setMenu] = useState(false);
  return (
    <div className="flex h-dvh overflow-hidden flex-col sm:flex-row">
      <div className="sm:hidden flex items-center justify-between px-4 h-14 shrink-0 border-b border-slate-800">
        <span className="font-semibold">Player2</span>
        <button
          aria-label={menu ? "Close navigation" : "Open navigation"}
          className="p-2"
          onClick={() => setMenu(!menu)}
        >
          {menu ? <X /> : <Menu />}
        </button>
      </div>
      <aside
        className={`${menu ? "flex" : "hidden"} sm:flex w-full sm:w-56 shrink-0 bg-slate-900 border-r border-slate-800 flex-col`}
      >
        <div className="hidden sm:flex h-20 items-center gap-3 px-5">
          <Sparkles className="text-teal-300" size={23} />
          <span className="font-semibold text-xl">Player2</span>
        </div>
        <nav className="flex-1 p-3 space-y-1" aria-label="Main navigation">
          {items.map((i) => (
            <NavLink
              key={i.to}
              to={i.to}
              onClick={() => setMenu(false)}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-3 rounded-lg text-sm ${isActive ? "bg-teal-400/10 text-teal-200" : "text-slate-300 hover:bg-slate-800"}`
              }
            >
              <i.icon size={18} />
              {i.label}
            </NavLink>
          ))}
          <details className="pt-4 text-sm text-slate-400">
            <summary className="cursor-pointer px-3 py-2">
              Advanced tools
            </summary>
            <NavLink
              to="/tickets"
              onClick={() => setMenu(false)}
              className="flex items-center gap-3 px-3 py-3"
            >
              <ListTodo size={17} />
              All tasks
            </NavLink>
            <NavLink
              to="/teams"
              onClick={() => setMenu(false)}
              className="flex items-center gap-3 px-3 py-3"
            >
              <Users size={17} />
              Teams
            </NavLink>
            <button
              onClick={() => {
                setTerminalOpen(!terminalOpen);
                setMenu(false);
              }}
              className="flex items-center gap-3 px-3 py-3"
            >
              <TerminalSquare size={17} />
              Terminal
            </button>
          </details>
        </nav>
        <p className="px-6 py-5 text-xs text-slate-400">
          Saved on this computer
        </p>
      </aside>
      <div className="flex-1 min-w-0 flex flex-col overflow-hidden">
        <main className="flex-1 overflow-auto bg-slate-950">
          <Outlet />
        </main>
        <TerminalPanel
          isOpen={terminalOpen}
          onClose={() => setTerminalOpen(false)}
        />
      </div>
    </div>
  );
}
