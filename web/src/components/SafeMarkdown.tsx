import Markdown from "react-markdown";
export default function SafeMarkdown({ children }: { children: string }) {
  return (
    <Markdown
      components={{
        img: ({ src, alt }) => (
          <a href={src} target="_blank" rel="noreferrer noopener">
            View image{alt ? `: ${alt}` : ""} (opens outside Player2)
          </a>
        ),
        a: ({ href, children }) => (
          <a href={href} target="_blank" rel="noreferrer noopener">
            {children}
          </a>
        ),
      }}
    >
      {children}
    </Markdown>
  );
}
