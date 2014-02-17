class FSFolder extends FSFile

  constructor: ->
    { @stack } = new Error
    super

  fetchContents:(dontWatch, callback)->
    [callback, dontWatch] = [dontWatch, callback]  unless callback?

    dontWatch ?= yes

    { treeController } = @getOptions()

    kite = @getKite()

    kite.vmStart()
    .then =>

      kite.fsReadDirectory
        path      : FSHelper.plainPath @path
        # onChange  : if dontWatch then null else (change) =>
        #   FSHelper.folderOnChange {
        #     @vmName
        #     @path
        #     change
        #     treeController
        #   }

    .then (response) =>
      files =
        if response?.files?
        then FSHelper.parseWatcher {
          @vmName
          parentPath: @path
          files: response.files
          treeController
        }
        else []

    .nodeify(callback)

    .then =>
      @emit 'fs.job.finished'


  save:(callback)->

    @emit "fs.save.started"

    @getKite().vmStart()

    .then =>
      @vmController.fsCreateDirectory {
        path: FSHelper.plainPath @path
      }

    .nodeify (err, response) ->
      callback null, response
      @emit "fs.save.finished", null, response

  saveAs:(callback)->
    log 'Not implemented yet.'
    callback? null

  remove:(callback)->
    @off 'fs.delete.finished'
    @on  'fs.delete.finished', =>
      return  unless finder = KD.getSingleton 'finderController'
      finder.stopWatching @path

    super callback, yes

  registerWatcher:(response)->
    {@stopWatching} = response
    finder = KD.getSingleton 'finderController'
    return unless finder
    finder.registerWatcher @path, @stopWatching  if @stopWatching
