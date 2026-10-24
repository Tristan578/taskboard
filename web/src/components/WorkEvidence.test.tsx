import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import WorkEvidence from './WorkEvidence';
import { api, type Ticket } from '../api/client';
vi.mock('../api/client', () => ({ api: {tickets: {complete: vi.fn(), resolve: vi.fn()}} }));
afterEach(() => {cleanup(); vi.clearAllMocks();});
const ticket = {id:'t1', title:'Task', status:'review', work:{agent:'Codex',sessionId:'one',summary:'Tests passed; awaiting merge',stale:false}} as Ticket;
describe('verified completion',()=>{
 it('keeps errors visible and does not signal completion when verification fails', async()=>{
  vi.mocked(api.tickets.complete).mockRejectedValue(new Error('Pull request is not merged'));
  const refresh=vi.fn();render(<WorkEvidence ticket={ticket} onRefresh={refresh}/>);
  await userEvent.type(screen.getByLabelText('Completion evidence URL'),'https://github.com/o/r/pull/1');
  await userEvent.click(screen.getByRole('button',{name:'Verify & mark Done'}));
  expect(await screen.findByRole('alert')).toHaveProperty('textContent','Pull request is not merged');
  expect(refresh).not.toHaveBeenCalled();
 });
 it('refreshes after verified completion',async()=>{
  vi.mocked(api.tickets.complete).mockResolvedValue({...ticket,status:'done'});
  const refresh=vi.fn();render(<WorkEvidence ticket={ticket} onRefresh={refresh}/>);
  await userEvent.type(screen.getByLabelText('Completion evidence URL'),'https://github.com/o/r/pull/1');
  await userEvent.click(screen.getByRole('button',{name:'Verify & mark Done'}));
  expect(api.tickets.complete).toHaveBeenCalledWith('t1','https://github.com/o/r/pull/1');expect(refresh).toHaveBeenCalledOnce();
 });
 it('shows both conflict choices and submits the explicit choice',async()=>{
  vi.mocked(api.tickets.resolve).mockResolvedValue({});
  render(<WorkEvidence ticket={{...ticket,work:{...ticket.work!,conflict:'Remote version'}}} onRefresh={vi.fn()}/>);
  await userEvent.click(screen.getByRole('button',{name:'Use GitHub version'}));
  expect(api.tickets.resolve).toHaveBeenCalledWith('t1','remote');
 });
});
