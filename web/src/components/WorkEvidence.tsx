import { useState } from "react";
import { api, type Ticket } from "../api/client";

export default function WorkEvidence({ ticket, onRefresh, manual = false }: {ticket: Ticket; onRefresh: () => void; manual?: boolean}) {
 const [evidence, setEvidence] = useState("");
 const [busy, setBusy] = useState(false);
 const [error, setError] = useState("");
 async function act(action: () => Promise<unknown>) {
  setBusy(true); setError("");
  try { await action(); onRefresh(); } catch(e) {setError(e instanceof Error ? e.message : "Action failed");} finally {setBusy(false);}
 }
 if(manual) return <section className="space-y-3 rounded-xl border border-slate-700 p-4"><h3 className="font-medium">Review the result</h3><p className="text-sm text-slate-300">You decide when this task is complete. Check the result before marking it done.</p>{ticket.status!=="done"?<button className="easy-button" disabled={busy} onClick={()=>act(()=>api.tickets.move(ticket.id,"done"))}>{busy?"Saving…":"Mark done"}</button>:<p className="text-teal-300">Marked done on your everyday board.</p>}{error&&<p role="alert" className="text-rose-300">{error}</p>}</section>;
 return <section className="space-y-3 rounded-xl border border-slate-700 bg-slate-950 p-4">
  <h3 className="text-sm font-semibold text-white">Work & evidence</h3>
  <p className="text-xs text-slate-400">{ticket.work?.agent ? `${ticket.work.agent} · ${ticket.work.stale ? "Activity stale" : "Recent activity"}` : "No agent activity reported"}</p>
  {ticket.work?.summary && <p className="text-sm text-slate-300">{ticket.work.summary}</p>}
  {ticket.work?.heartbeatAt && <p className="text-xs text-slate-500">Last report: {new Date(ticket.work.heartbeatAt).toLocaleString()}</p>}
  {ticket.work?.completionUrl ? <a className="text-sm text-emerald-400 underline" href={ticket.work.completionUrl} target="_blank" rel="noreferrer">Verified completion evidence ↗</a> : <p className="text-xs text-slate-400">Done requires a verified merged PR or successful GitHub deployment. Legacy Done tickets may have no recorded evidence.</p>}
  {ticket.work?.conflict && <div className="space-y-2 rounded-lg border border-amber-700 p-3">
   <p className="text-sm text-amber-300">GitHub conflict — both versions preserved</p>
   <details className="text-xs text-slate-400"><summary>Inspect GitHub version before resolving</summary><pre className="whitespace-pre-wrap max-h-48 overflow-auto">{ticket.work.conflict}</pre></details>
   <div className="flex gap-2">
    <button disabled={busy} onClick={() => act(() => api.tickets.resolve(ticket.id,"local"))} className="rounded bg-slate-800 px-3 py-2 text-xs">Keep local version</button>
    <button disabled={busy} onClick={() => act(() => api.tickets.resolve(ticket.id,"remote"))} className="rounded bg-slate-800 px-3 py-2 text-xs">Use GitHub version</button>
   </div>
  </div>}
  <label className="block text-xs text-slate-400">Completion evidence for this task
   <input aria-label="Completion evidence URL" value={evidence} onChange={e=>setEvidence(e.target.value)} placeholder="https://github.com/owner/repo/pull/123" className="mt-2 w-full rounded border border-slate-700 bg-slate-900 p-2 text-sm" />
  </label>
  <button disabled={busy || !evidence.trim()} onClick={() => act(() => api.tickets.complete(ticket.id,evidence.trim()))} className="rounded bg-emerald-600 px-3 py-2 text-xs font-semibold text-white disabled:opacity-40">{busy ? "Checking…" : "Verify & mark Done"}</button>
  {error && <p role="alert" className="text-xs text-red-400">{error}</p>}
 </section>;
}
