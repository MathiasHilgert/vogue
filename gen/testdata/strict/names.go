package place

// AlternateName is another name a place is known by. It is declared in a
// second file of the package, so the generated tests of both files share the
// examples of the same rules: the fixture that proves strings repeated across
// generated files are named without colliding.
//vogue:string AlternateName squish required min=1 max=200

// LanguageCode is the ISO 639-1 code of the language an alternate name is in.
//vogue:string LanguageCode trim lower required len=2 example=es
