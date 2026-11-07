import ReactMarkdown, { type Components } from "react-markdown";
import rehypeKatex from "rehype-katex";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import { cn } from "@/lib/ui";

const inlineComponents: Components = {
  p: ({ children }) => <>{children}</>,
};

/**
 * remark-math only treats `$$` as display math when the fences are on their
 * own lines. Content authors (and our AI Content Admin) often write
 * `$$ x $$` on one line, so split such lines into fenced blocks.
 */
export function normalizeMath(md: string): string {
  return md.replace(/^([ \t]*)\$\$([^\n]+?)\$\$[ \t]*$/gm, (_m, indent: string, body: string) => `${indent}$$\n${indent}${body.trim()}\n${indent}$$`);
}

type Props = {
  children: string | null | undefined;
  className?: string;
  /** Render without block wrappers (for choices / short labels). */
  inline?: boolean;
};

/**
 * Markdown with KaTeX math ($…$ and $$…$$). Raw HTML is never rendered
 * (skipHtml) and unsafe URLs are stripped by react-markdown's default
 * urlTransform.
 */
export function Markdown({ children, className, inline = false }: Props) {
  const content = (
    <ReactMarkdown
      remarkPlugins={[remarkGfm, remarkMath]}
      rehypePlugins={[[rehypeKatex, { throwOnError: false, strict: "ignore", output: "htmlAndMathml" }]]}
      skipHtml
      components={inline ? inlineComponents : undefined}
    >
      {inline ? (children ?? "") : normalizeMath(children ?? "")}
    </ReactMarkdown>
  );
  if (inline) return <span className={className}>{content}</span>;
  return <div className={cn("prose-lesson", className)}>{content}</div>;
}
