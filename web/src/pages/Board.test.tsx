import {afterEach, expect, it, vi} from 'vitest';
import {cleanup, render, screen} from '@testing-library/react';
import Board from './Board';
import {api, type Ticket} from '../api/client';
vi.mock('../api/client',()=>({api:{board:{get:vi.fn()},projects:{list:vi.fn()},teams:{list:vi.fn()},sync:{status:vi.fn(),retry:vi.fn()}}}));
afterEach(()=>{cleanup();vi.clearAllMocks();});
function setup(){
 vi.mocked(api.projects.list).mockResolvedValue([]);vi.mocked(api.teams.list).mockResolvedValue([]);
 vi.mocked(api.sync.status).mockResolvedValue({pendingJobs:0,failedJobs:0,githubConfigured:false});
}
it('surfaces board failure instead of claiming the backlog loaded empty',async()=>{
 setup();vi.mocked(api.board.get).mockRejectedValue(new Error('Database unavailable'));
 render(<Board/>);expect((await screen.findByRole('alert')).textContent).toContain('Database unavailable');
});
it('shows agent summaries and stale work from the shared board',async()=>{
 setup();vi.mocked(api.board.get).mockResolvedValue({projectId:'p1',columns:[{status:'in_progress',tickets:[{id:'t1',projectId:'p1',title:'Implement workflow',status:'in_progress',priority:'high',work:{agent:'Codex',sessionId:'run',summary:'Investigating sync',stale:true}} as Ticket]}]});
 render(<Board/>);
 expect((await screen.findByText('Codex · No recent heartbeat')).textContent).toContain('Codex');
 expect(screen.getByText('1 stale / untracked')).toBeTruthy();expect(screen.getByText('Investigating sync')).toBeTruthy();
});
