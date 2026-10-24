export interface Project {
  completionPolicy?: "manual" | "verified";
  id: string;
  name: string;
  prefix: string;
  description: string;
  icon: string;
  color: string;
  status: string;
  githubRepo?: string;
  strict: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface Team {
  id: string;
  name: string;
  color: string;
  createdAt: string;
}

export interface Label {
  id: string;
  name: string;
  color: string;
}

export interface Subtask {
  id: string;
  ticketId: string;
  title: string;
  completed: boolean;
  position: number;
}

export interface WorkState {
 agent: string; sessionId: string; heartbeatAt?: string; summary: string;
 stale: boolean; completionUrl?: string; completedAt?: string; conflict?: string;
}
export interface SyncStatus { lastError?: string; pendingJobs: number; failedJobs: number; lastSyncAt?: string; githubConfigured?: boolean; }
export type TicketUpdate = Partial<Ticket> & { expectedRevision?: number };
export interface Ticket {
 repeatRule?: string;
 repeatAnchor?: number;
 repeatSourceId?: string;
 revision: number;
 work?: WorkState;
  id: string;
  projectId: string;
  teamId?: string;
  number: number;
  title: string;
  description: string;
  status: string;
  priority: string;
  dueDate?: string;
  position: number;
  githubIssueNumber?: number;
  userStory?: string;
  acceptanceCriteria?: string;
  technicalDetails?: string;
  testingDetails?: string;
  isDraft: boolean;
  createdAt: string;
  updatedAt: string;
  projectPrefix: string;
  labels: Label[];
  subtasks: Subtask[];
  blockedBy: string[];
}

export interface BoardColumn {
  status: string;
  tickets: Ticket[];
}

export interface Board {
  projectId: string;
  columns: BoardColumn[];
}

export interface PaginatedResult<T> {
  data: T[];
  total: number;
  limit: number;
  offset: number;
  hasMore: boolean;
}

export interface PaginationParams {
  limit?: number;
  offset?: number;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function buildQuery(params: any): string {
  const entries = Object.entries(params).filter(([, v]) => v !== undefined && v !== "");
  if (entries.length === 0) return "";
  return "?" + entries.map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`).join("&");
}

export class ApiError extends Error { status: number; constructor(message:string,status:number){super(message);this.status=status;} }

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) {
    const text = await res.text();
    let message = text;
    try { message = JSON.parse(text).error || text; } catch { /* Plain text response. */ }
    throw new ApiError(message || "Something went wrong. Please try again.",res.status);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

export interface SavedBackup { id:string; createdAt:string; bytes:number; reason:string }
export interface RecoveryStatus { backups:SavedBackup[]; lastError?:string; directory:string }
export interface RecoveryPreview { id:string; boards:string[]; tasks:number }
export interface DeletedTask {id:string;title:string;board:string;deletedAt:string;revision:number;linked:boolean}
export const api = {
 recovery: {
  cancelPreview:(id:string)=>request<void>(`/api/recovery/previews/${id}`,{method:'DELETE'}),
  status:()=>request<RecoveryStatus>('/api/recovery'),
  create:()=>request<SavedBackup>('/api/recovery/backups',{method:'POST'}),
  preview:(file:File)=>request<RecoveryPreview>('/api/recovery/preview',{method:'POST',headers:{'Content-Type':'application/octet-stream'},body:file}),
  previewSaved:(id:string)=>request<RecoveryPreview>(`/api/recovery/backups/${encodeURIComponent(id)}/preview`,{method:'POST'}),
  restore:(id:string)=>request<{projectIds:string[]}>('/api/recovery/restore',{method:'POST',body:JSON.stringify({id,confirm:true})}),
  trash:()=>request<DeletedTask[]>('/api/trash'),
  undo:(id:string,revision:number)=>request<Ticket>(`/api/trash/${id}/restore`,{method:'POST',body:JSON.stringify({revision})}),
 },
 connections: () => request<{mcpServers:Record<string,{command:string;args:string[]}>}>("/api/connections"),
 welcome: { create: (data: {requestId:string;name:string;goal:string;completionPolicy:"manual"|"verified";tasks:{title:string;description:string}[]}) => request<Project>("/api/welcome", {method:"POST",body:JSON.stringify(data)}) },
 sync: {
  status: () => request<SyncStatus>("/api/sync/status"),
  retry: () => request("/api/sync/retry", {method:"POST"}),
 },
  projects: {
    list: () => request<Project[]>("/api/projects"),
    listPaginated: (params: PaginationParams & { status?: string }) =>
      request<PaginatedResult<Project>>(`/api/projects${buildQuery(params)}`),
    get: (id: string) => request<Project>(`/api/projects/${id}`),
    create: (data: Partial<Project>) =>
      request<Project>("/api/projects", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (id: string, data: Partial<Project>) =>
      request<Project>(`/api/projects/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    delete: (id: string) =>
      request<void>(`/api/projects/${id}`, { method: "DELETE" }),
  },

  teams: {
    list: () => request<Team[]>("/api/teams"),
    listPaginated: (params: PaginationParams) =>
      request<PaginatedResult<Team>>(`/api/teams${buildQuery(params)}`),
    get: (id: string) => request<Team>(`/api/teams/${id}`),
    create: (data: Partial<Team>) =>
      request<Team>("/api/teams", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (id: string, data: Partial<Team>) =>
      request<Team>(`/api/teams/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    delete: (id: string) =>
      request<void>(`/api/teams/${id}`, { method: "DELETE" }),
  },

  tickets: {
    complete: (id: string, evidence: string, expectedRevision?: number) => request<Ticket>(`/api/tickets/${id}/complete`, {method:"POST",body:JSON.stringify({evidence,expectedRevision})}),
    resolve: (id: string, choice: "local" | "remote") => request(`/api/tickets/${id}/resolve`, {method:"POST",body:JSON.stringify({choice})}),
    list: () => request<Ticket[]>("/api/tickets"),
    listPaginated: (params: PaginationParams & { projectId?: string; status?: string; priority?: string }) =>
      request<PaginatedResult<Ticket>>(`/api/tickets${buildQuery(params)}`),
    get: (id: string) => request<Ticket>(`/api/tickets/${id}`),
    create: (data: Partial<Ticket>) =>
      request<Ticket>("/api/tickets", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (id: string, data: TicketUpdate) =>
      request<Ticket>(`/api/tickets/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    delete: async (id: string, revision?: number) => {
      await request<void>(`/api/tickets/${id}${revision===undefined?'':`?revision=${revision}`}`, { method: "DELETE" });
      if(revision!==undefined) window.dispatchEvent(new CustomEvent('player2:deleted',{detail:{id,revision:revision+1}}));
    },
    move: (id: string, status: string, position?: number, expectedRevision?: number) =>
      request<Ticket>(`/api/tickets/${id}/move`, {
        method: "POST",
        body: JSON.stringify({ status, position, expectedRevision }),
      }),
    addSubtask: (id: string, title: string) =>
      request<Subtask>(`/api/tickets/${id}/subtasks`, {
        method: "POST",
        body: JSON.stringify({ title }),
      }),
  },

  subtasks: {
    toggle: (id: string) =>
      request<Subtask>(`/api/subtasks/${id}/toggle`, { method: "POST" }),
    delete: (id: string) =>
      request<void>(`/api/subtasks/${id}`, { method: "DELETE" }),
  },

  labels: {
    list: () => request<Label[]>("/api/labels"),
    listPaginated: (params: PaginationParams) =>
      request<PaginatedResult<Label>>(`/api/labels${buildQuery(params)}`),
    create: (data: Partial<Label>) =>
      request<Label>("/api/labels", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (id: string, data: Partial<Label>) =>
      request<Label>(`/api/labels/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    delete: (id: string) =>
      request<void>(`/api/labels/${id}`, { method: "DELETE" }),
  },

  board: {
    get: (projectId?: string) =>
      request<Board>(`/api/board${projectId ? `?projectId=${projectId}` : ""}`),
  },
};
