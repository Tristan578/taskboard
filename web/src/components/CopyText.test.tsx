import { afterEach, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import CopyText from './CopyText';
afterEach(()=>{cleanup();vi.restoreAllMocks();});
it('keeps a selectable copy when clipboard permission is denied',async()=>{
 const user=userEvent.setup();vi.spyOn(navigator.clipboard,'writeText').mockRejectedValue(new Error('Permission denied'));
 render(<CopyText text="Help me plan next week"/>);await user.click(screen.getByRole('button',{name:'Copy instructions'}));
 expect(screen.getByRole('status').textContent).toContain('Select the text below');
 await user.click(screen.getByText('Review or select the text'));expect(screen.getByLabelText('Instructions to copy')).toHaveProperty('value','Help me plan next week');
});
