function sep1(rule, separator) {
  return seq(rule, repeat(seq(separator, rule)));
}

module.exports = grammar({
  name: 'vexil',

  extras: $ => [
    /[ \t]/,
  ],

  conflicts: $ => [
    [$.media_alt, $.link_label],
  ],

  rules: {
    document: $ => repeat(
      choice(
        $._block,
        $._blank_line,
      ),
    ),

    _block: $ => choice(
      $.meta_directive,
      $.heading,
      $.divider,
      $.code_fence,
      $.table,
      $.callout,
      $.blockquote,
      $.unordered_list_item,
      $.ordered_list_item,
      $.task_list_item,
      $.paragraph,
    ),

    meta_directive: $ => seq(
      '@',
      field('key', $.identifier),
      optional(seq(
        ':',
        field('value', $.meta_value),
      )),
      $._newline,
    ),

    identifier: $ => /[a-zA-Z0-9_.-]+/,

    meta_value: $ => /[^\r\n]+/,

    heading: $ => seq(
      field('level', $.heading_marker),
      repeat($._inline),
      $._newline,
    ),

    heading_marker: $ => choice('#', '##', '###', '####'),

    divider: $ => seq(
      '---',
      $._newline,
    ),

    code_fence: $ => seq(
      '```',
      optional(choice(
        seq(
          field('language', $.identifier),
          optional(choice(
            seq(':', field('identifier', $.identifier)),
            field('identifier', $.identifier),
          )),
        ),
      )),
      $._newline,
      optional(field('content', $.code_fence_content)),
      '```',
      $._newline,
    ),

    code_fence_content: $ => repeat1(
      choice(
        $._code_line,
        $._blank_line,
      ),
    ),

    _code_line: $ => seq(
      token(prec(1, /(?:`{1,2}[^`\r\n]|[^`\r\n])[^\r\n]*/)),
      $._newline,
    ),

    table: $ => seq(
      $.table_header,
      repeat($.table_row),
      '::',
      $._newline,
    ),

    table_header: $ => seq(
      '::table',
      '[',
      sep1(alias($._table_header_cell, $.table_cell), choice('|', ',')),
      ']',
      $._newline,
    ),

    table_row: $ => seq(
      '-',
      sep1(alias($._table_row_cell, $.table_cell), '|'),
      $._newline,
    ),

    _table_header_cell: $ => choice(
      token(prec(-1, /[^|,\]\r\n*`_~\[\\]+/)),
      repeat1($._inline_table_header),
    ),

    _table_row_cell: $ => choice(
      token(prec(-1, /(?:\\\||[^|\r\n*`_~\[\\])+/)),
      repeat1($._inline_table_row),
    ),

    callout: $ => prec.right(seq(
      '!',
      field('type', $.callout_type),
      optional(seq(
        ':',
        field('title', $.callout_title),
      )),
      $._newline,
      repeat(seq(repeat1($._inline), $._newline)),
    )),

    callout_type: $ => /[a-zA-Z0-9_-]+/,

    callout_title: $ => choice(
      token(prec(-1, /[^\r\n*`_~\[\\]+/)),
      repeat1($._inline),
    ),

    blockquote: $ => prec.right(repeat1(
      seq(
        '>',
        repeat($._inline),
        $._newline,
      ),
    )),

    unordered_list_item: $ => seq(
      '-',
      repeat($._inline),
      $._newline,
    ),

    ordered_list_item: $ => seq(
      choice(/\d+\./, '.'),
      repeat($._inline),
      $._newline,
    ),

    task_list_item: $ => seq(
      field('status', $.task_status),
      repeat($._inline),
      $._newline,
    ),

    task_status: $ => choice('[ ]', '[x]', '[X]'),

    paragraph: $ => prec(-1, seq(
      repeat1($._inline),
      $._newline,
    )),

    // --- Inline Rules ---

    _inline: $ => choice(
      $.escape_sequence,
      $.strong,
      $.emphasis,
      $.inline_code,
      $.strikethrough,
      $.media,
      $.link,
      $.text,
    ),

    _inline_nested: $ => choice(
      $.escape_sequence,
      $.emphasis,
      $.inline_code,
      $.strikethrough,
      $.media,
      $.link,
      $.text,
    ),

    _inline_emphasis: $ => choice(
      $.escape_sequence,
      $.strong,
      $.inline_code,
      $.strikethrough,
      $.media,
      $.link,
      $.text,
    ),

    _inline_strikethrough: $ => choice(
      $.escape_sequence,
      $.strong,
      $.emphasis,
      $.inline_code,
      $.media,
      $.link,
      $.text,
    ),

    _inline_link: $ => choice(
      $.escape_sequence,
      $.strong,
      $.emphasis,
      $.inline_code,
      $.strikethrough,
      alias($._link_text, $.text),
    ),

    _link_text: $ => token(prec(-1, /[^\\*_`~\[\]\r\n]+/)),

    _inline_table_header: $ => choice(
      $.escape_sequence,
      $.strong,
      $.emphasis,
      $.inline_code,
      $.strikethrough,
      $.media,
      $.link,
      alias($._table_header_text, $.text),
    ),

    _table_header_text: $ => token(prec(-1, /[^|,\]\\*_`~\[\r\n]+/)),

    _inline_table_row: $ => choice(
      $.escape_sequence,
      $.strong,
      $.emphasis,
      $.inline_code,
      $.strikethrough,
      $.media,
      $.link,
      alias($._table_row_text, $.text),
    ),

    _table_row_text: $ => token(prec(-1, /(?:\\\||[^|\\*_`~\[\r\n])+/)),

    strong: $ => seq(
      '*',
      repeat1($._inline_nested),
      '*',
    ),

    emphasis: $ => seq(
      '_',
      repeat1($._inline_emphasis),
      '_',
    ),

    inline_code: $ => seq(
      '`',
      token(prec(1, /[^`\r\n]+/)),
      '`',
    ),

    strikethrough: $ => seq(
      '~',
      repeat1($._inline_strikethrough),
      '~',
    ),

    escape_sequence: $ => token(seq(
      '\\',
      choice(
        '*', '_', '`', '~', '|', '[', ']', '(', ')', '\\', '!', '-', '>', '#'
      ),
    )),

    media: $ => prec.dynamic(1, seq(
      '[',
      field('alt', $.media_alt),
      ']',
      '(',
      field('source', $.media_source),
      ')',
    )),

    media_alt: $ => repeat1($._inline_link),

    media_source: $ => token(prec(1, choice(
      seq('img:', /[^\r\n)]*/),
      /[^\r\n)]+\.(?:png|PNG|jpg|JPG|jpeg|JPEG|svg|SVG|webp|WEBP|gif|GIF|mp4|MP4)(?:[?#][^\r\n)]*)?/,
    ))),

    link: $ => seq(
      '[',
      field('label', $.link_label),
      ']',
      '(',
      field('destination', $.link_destination),
      ')',
    ),

    link_label: $ => repeat1($._inline_link),

    link_destination: $ => token(prec(1, /[^\r\n)]+/)),

    text: $ => token(prec(-1, /[^\\*_`~\[\r\n]+/)),

    _blank_line: $ => $._newline,

    _newline: $ => /\r?\n/,
  },
});
