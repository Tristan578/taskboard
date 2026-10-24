import { useState } from "react";
export default function CopyText({
  text,
  label = "Copy instructions",
}: {
  text: string;
  label?: string;
}) {
  const [message, setMessage] = useState("");
  async function copy() {
    try {
      await navigator.clipboard.writeText(text);
      setMessage("Copied. Paste this into your AI chat.");
    } catch {
      setMessage(
        "Copy is unavailable here. Select the text below and copy it yourself.",
      );
    }
  }
  return (
    <div className="space-y-2">
      <button type="button" className="easy-button" onClick={copy}>
        {label}
      </button>
      <p role="status" className="text-sm text-teal-300">
        {message}
      </p>
      <details>
        <summary className="cursor-pointer text-sm text-slate-300">
          Review or select the text
        </summary>
        <textarea
          aria-label="Instructions to copy"
          readOnly
          value={text}
          rows={8}
          className="easy-input mt-3"
          onFocus={(e) => e.target.select()}
        />
      </details>
    </div>
  );
}
