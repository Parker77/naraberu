// Safe Markdown subset for Synopsis / Review notes.
// Builds DOM via createElement + textContent only—never injects user HTML.
// Subset: h1–h3, bold, italic, strikethrough, inline code, flat lists,
// blockquotes, paragraphs, hard line breaks. No links, no code blocks,
// no horizontal rules, no h4–h6.

function renderMarkdown(src) {
    const frag = document.createDocumentFragment();
    const text = String(src || '').replace(/\r\n/g, '\n');
    const lines = text.split('\n');
    let i = 0;

    while (i < lines.length) {
        const line = lines[i];
        if (/^\s*$/.test(line)) {
            i++;
            continue;
        }

        const heading = line.match(/^(#{1,3})\s+(.*)$/);
        if (heading) {
            const el = document.createElement('h' + heading[1].length);
            appendMarkdownInline(el, heading[2].trim());
            frag.appendChild(el);
            i++;
            continue;
        }

        if (/^>\s?/.test(line)) {
            const quoteLines = [];
            while (i < lines.length && /^>\s?/.test(lines[i])) {
                quoteLines.push(lines[i].replace(/^>\s?/, ''));
                i++;
            }
            const bq = document.createElement('blockquote');
            appendMarkdownParagraph(bq, quoteLines);
            frag.appendChild(bq);
            continue;
        }

        if (/^\s*[-*+]\s+/.test(line)) {
            const ul = document.createElement('ul');
            while (i < lines.length && /^\s*[-*+]\s+/.test(lines[i])) {
                const li = document.createElement('li');
                appendMarkdownInline(li, lines[i].replace(/^\s*[-*+]\s+/, ''));
                ul.appendChild(li);
                i++;
            }
            frag.appendChild(ul);
            continue;
        }

        if (/^\s*\d+\.\s+/.test(line)) {
            const ol = document.createElement('ol');
            while (i < lines.length && /^\s*\d+\.\s+/.test(lines[i])) {
                const li = document.createElement('li');
                appendMarkdownInline(li, lines[i].replace(/^\s*\d+\.\s+/, ''));
                ol.appendChild(li);
                i++;
            }
            frag.appendChild(ol);
            continue;
        }

        const paraLines = [];
        while (i < lines.length && !/^\s*$/.test(lines[i]) && !isMarkdownBlockStart(lines[i])) {
            paraLines.push(lines[i]);
            i++;
        }
        if (paraLines.length) {
            const p = document.createElement('p');
            appendMarkdownParagraph(p, paraLines);
            frag.appendChild(p);
        }
    }

    return frag;
}

function isMarkdownBlockStart(line) {
    return /^(#{1,3})\s+/.test(line) ||
        /^>/.test(line) ||
        /^\s*[-*+]\s+/.test(line) ||
        /^\s*\d+\.\s+/.test(line);
}

function appendMarkdownParagraph(parent, lines) {
    lines.forEach((line, idx) => {
        if (idx > 0) parent.appendChild(document.createElement('br'));
        appendMarkdownInline(parent, line);
    });
}

function appendMarkdownInline(parent, text) {
    let i = 0;
    let plain = '';
    const flush = () => {
        if (plain) {
            parent.appendChild(document.createTextNode(plain));
            plain = '';
        }
    };
    const s = String(text || '');

    while (i < s.length) {
        if (s[i] === '`') {
            const end = s.indexOf('`', i + 1);
            if (end !== -1) {
                flush();
                const code = document.createElement('code');
                code.textContent = s.slice(i + 1, end);
                parent.appendChild(code);
                i = end + 1;
                continue;
            }
        }

        if (s.startsWith('**', i) || s.startsWith('__', i)) {
            const marker = s.slice(i, i + 2);
            const end = s.indexOf(marker, i + 2);
            if (end !== -1) {
                flush();
                const strong = document.createElement('strong');
                appendMarkdownInline(strong, s.slice(i + 2, end));
                parent.appendChild(strong);
                i = end + 2;
                continue;
            }
        }

        if (s.startsWith('~~', i)) {
            const end = s.indexOf('~~', i + 2);
            if (end !== -1) {
                flush();
                const del = document.createElement('del');
                appendMarkdownInline(del, s.slice(i + 2, end));
                parent.appendChild(del);
                i = end + 2;
                continue;
            }
        }

        if (s[i] === '*' && s[i + 1] !== '*') {
            const end = s.indexOf('*', i + 1);
            if (end !== -1 && s[end + 1] !== '*') {
                flush();
                const em = document.createElement('em');
                appendMarkdownInline(em, s.slice(i + 1, end));
                parent.appendChild(em);
                i = end + 1;
                continue;
            }
        }

        if (s[i] === '_' && s[i + 1] !== '_') {
            const end = s.indexOf('_', i + 1);
            if (end !== -1 && s[end + 1] !== '_') {
                flush();
                const em = document.createElement('em');
                appendMarkdownInline(em, s.slice(i + 1, end));
                parent.appendChild(em);
                i = end + 1;
                continue;
            }
        }

        plain += s[i];
        i++;
    }
    flush();
}
