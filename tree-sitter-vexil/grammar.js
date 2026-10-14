function sep1(rule, separator) {
  return seq(rule, repeat(seq(separator, rule)));
}

module.exports = grammar({
  name: 'vexil',

  extras: $ => [
    /[ \t]/,
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
      optional(field('content', $.text)),
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

    _table_header_cell: $ => token(prec(1, /[^|,\]\r\n]+/)),

    _table_row_cell: $ => token(prec(1, /(?:\\\||[^|\r\n])+/)),

    callout: $ => prec.right(seq(
      '!',
      field('type', $.callout_type),
      optional(seq(
        ':',
        field('title', $.callout_title),
      )),
      $._newline,
      repeat(seq($.text, $._newline)),
    )),

    callout_type: $ => /[a-zA-Z0-9_-]+/,

    callout_title: $ => /[^\r\n]+/,

    blockquote: $ => prec.right(repeat1(
      seq(
        '>',
        optional($.text),
        $._newline,
      ),
    )),

    unordered_list_item: $ => seq(
      '-',
      optional($.text),
      $._newline,
    ),

    ordered_list_item: $ => seq(
      choice(/\d+\./, '.'),
      optional($.text),
      $._newline,
    ),

    task_list_item: $ => seq(
      field('status', $.task_status),
      optional($.text),
      $._newline,
    ),

    task_status: $ => choice('[ ]', '[x]', '[X]'),

    paragraph: $ => prec(-1, seq(
      $.text,
      $._newline,
    )),

    text: $ => token(prec(-1, /[^\r\n]+/)),

    _blank_line: $ => $._newline,

    _newline: $ => /\r?\n/,
  },
});
