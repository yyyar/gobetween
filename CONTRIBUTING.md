# How to contribute

## Reporting bugs and proposals

You can [submit an issue](https://github.com/yyyar/gobetween/issues/new). Please be descriptive and provide as much evidence 
as you can, so that we could either help you to fix your setup, or reproduce the issue on our side. This includes config of 
gobetween, logs, expected behavior and actual behavior, steps to reproduce. 

Please make sure you've cleaned the data you provide us from sensitive information such as public ips, user names, emails, logins, 
security tokens and other information that you don't want to make public.

## Submitting changes

Please send a [GitHub Pull Request to gobetween](https://github.com/yyyar/gobetween/pull/new/master) with a clear description 
of what you've done. Please make sure all your commits are atomic (one feature per commit). Please follow our coding conventions

  * Prior to submitting PR, make an issue with a question. It could save a lot of time and efforts because we could be already
  working on it, or the change you propose does not fit the gobetween ideology.
  * In case if you have intermediate commits, such as "WIP - work in progress" or sequence of commits that add some files/code and 
  then removes it, please [squash](https://git-scm.com/book/en/v2/Git-Tools-Rewriting-History) them prior to pull request
  * Please make sure git commits are descriptive
  * Please change just a required minimum of code in order to implement your feature or fix a bug. Pull requests that have 
  a lot of not related changes, are hard to review and hard to merge due to possible conflicts with other branches:
    * Don't rename existing variables if it's not required for your change
    * Don't add or remove empty strings
    * Don't autoformat files according to your IDE settings, leave original formatting in places that you don't explicitly change
    * Write go-doc style comments on functions you add
  
  If your not related change is really good -- make a distinct pull request for it.
  

## Coding conventions

Please read the code and you'll get the essence of it. Following coding conventions is an attempt to make a short but not 
complete extract:

  * Simple code is better than smart tricky code
  * Keep performance in mind - every line your write could be hit millions times per second
  * Please use [gofmt](https://golang.org/cmd/gofmt/), it's easy to setup gofmt integration with your editor
  * Error values should be handled (and wrapped adding description using [fmt.Errorf](https://golang.org/pkg/fmt/#Errorf)
  * Don't panic() unless it's an assert to detect logically impossible situations
  * We have borrowed 'this' as the name of a struct's method receiver, in order to survive possible struct rename. You can use
  both options - 'this' and 'x' where x is the first letter of a struct name.
  * Please use common sense
  * Have fun! :)
  
