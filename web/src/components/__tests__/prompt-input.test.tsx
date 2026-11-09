/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import { toast } from 'sonner'
import { afterEach, expect, it, vi } from 'vitest'

import {
  PromptInput,
  PromptInputProvider,
  PromptInputTextarea,
  usePromptInputAttachments,
} from '@/components/ai-elements/prompt-input'

function AttachmentCount() {
  const { files } = usePromptInputAttachments()
  return <output aria-label='Attachments'>{files.length}</output>
}

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

it('reports attachment conversion failure and retains the message for a successful retry', async () => {
  vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:test-attachment')
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
  const fetchFile = vi
    .fn()
    .mockRejectedValueOnce(new Error('Attachment unavailable'))
    .mockResolvedValue({
      blob: async () => new Blob(['hello'], { type: 'text/plain' }),
    })
  vi.stubGlobal('fetch', fetchFile)
  const showError = vi.spyOn(toast, 'error').mockImplementation(() => 'error')
  const submit = vi.fn()
  render(
    <PromptInputProvider initialInput='Please read this file'>
      <PromptInput onSubmit={submit}>
        <PromptInputTextarea />
        <AttachmentCount />
        <button type='submit'>Send</button>
      </PromptInput>
    </PromptInputProvider>
  )
  fireEvent.change(screen.getByLabelText('Upload files'), {
    target: {
      files: [new File(['hello'], 'hello.txt', { type: 'text/plain' })],
    },
  })
  fireEvent.click(screen.getByRole('button', { name: 'Send' }))
  await waitFor(() =>
    expect(showError).toHaveBeenCalledWith('Attachment unavailable')
  )
  expect(submit).not.toHaveBeenCalled()
  expect(screen.getByRole('textbox')).toHaveValue('Please read this file')
  expect(screen.getByLabelText('Attachments')).toHaveTextContent('1')

  fireEvent.click(screen.getByRole('button', { name: 'Send' }))
  await waitFor(() =>
    expect(submit).toHaveBeenCalledWith(
      {
        text: 'Please read this file',
        files: [
          {
            type: 'file',
            filename: 'hello.txt',
            mediaType: 'text/plain',
            url: 'data:text/plain;base64,aGVsbG8=',
          },
        ],
      },
      expect.anything()
    )
  )
  await waitFor(() =>
    expect(screen.getByLabelText('Attachments')).toHaveTextContent('0')
  )
  expect(screen.getByRole('textbox')).toHaveValue('')
})
