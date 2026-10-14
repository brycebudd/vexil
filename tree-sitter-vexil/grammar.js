module.exports = grammar({
  name: 'vexil',

  extras: $ => [
    /[ \t]/,
  ],

  rules: {
    document: $ => repeat(
      choice(
        $.paragraph,
        $._blank_line,
      ),
    ),

    paragraph: $ => seq(
      $.text,
      $._newline,
    ),

    text: $ => /[^\r\n]+/,

    _blank_line: $ => $._newline,

    _newline: $ => /\r?\n/,
  },
});
