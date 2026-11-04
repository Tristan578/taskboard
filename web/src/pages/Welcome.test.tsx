import { afterEach, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import Welcome from './Welcome';
import { api, type Project } from '../api/client';
import { parseTaskLines } from '../lib/plans';
vi.mock('../api/client',()=>({api:{welcome:{create:vi.fn()}}}));
afterEach(()=>{cleanup();vi.clearAllMocks();});
function setup(){render(<MemoryRouter><Routes><Route path="/" element={<Welcome/>}/><Route path="/board" element={<p>Your new board</p>}/></Routes></MemoryRouter>);}
it('creates an everyday board without a repository, key, prefix, or agent configuration',async()=>{
 vi.mocked(api.welcome.create).mockResolvedValue({id:'new-board'} as Project);setup();
 await userEvent.click(screen.getByRole('button',{name:/Organize my household/}));
 await userEvent.click(screen.getByRole('button',{name:/Shape my plan/}));
 expect(screen.getByLabelText(/Tasks to add/)).toHaveProperty('value',expect.stringContaining('appointments'));
 await userEvent.click(screen.getByRole('button',{name:/Create my board/}));
 expect(await screen.findByText('Your new board')).toBeTruthy();
 expect(api.welcome.create).toHaveBeenCalledWith(expect.objectContaining({name:'Organize my household',completionPolicy:'manual',tasks:expect.any(Array)}));
});
it('keeps the plan and request ID for a safe retry after save failure',async()=>{
 vi.mocked(api.welcome.create).mockRejectedValueOnce(new Error('Connection interrupted')).mockResolvedValueOnce({id:'saved'} as Project);setup();
 await userEvent.click(screen.getByRole('button',{name:/Ship software or infrastructure/}));await userEvent.click(screen.getByRole('button',{name:/Shape my plan/}));
 await userEvent.click(screen.getByRole('button',{name:/Create my board/}));expect(await screen.findByRole('alert')).toHaveProperty('textContent','Connection interrupted');
 await userEvent.click(screen.getByRole('button',{name:/Create my board/}));
 const calls=vi.mocked(api.welcome.create).mock.calls;expect(calls[0][0]).toEqual(calls[1][0]);expect(calls[0][0].completionPolicy).toBe('verified');
});
it('accepts numbered AI lists as reviewed task titles without executing their contents',()=>{expect(parseTaskLines('1. Make a plan\n- Try it\n\n2) Review')).toEqual([{title:'Make a plan',description:''},{title:'Try it',description:''},{title:'Review',description:''}]);});
