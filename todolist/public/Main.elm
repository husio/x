import Html
import Html.Events
import Html.Attributes
import Signal


type alias ID = Int

type Action
    = NoOp
    | Add String
    | Remove ID


type alias Model =
    { entries : List Entry
    , input : String
    }

type alias Entry =
    { id : ID
    , content : String
    }


actions : Signal.Mailbox Action
actions =
    Signal.mailbox NoOp


model : Model
model =
    { entries = [], input = "" }

update : Action -> Model -> Model
update action model =
    case action of
        NoOp ->
            model
        Add content ->
            { model
            | entries = {id = 0, content = content} :: model.entries
            }
        Remove id ->
            { model | entries = [] }

input =
    let
        keyAction key =
            if key == 13 then Add "bobbins" else NoOp
        onKeyPress = Html.Events.onKeyDown actions.address keyAction
    in
        Html.input
        [ Html.Attributes.placeholder "content"
        , Html.Events.on "input" Html.Events.targetValue (\txt -> Signal.message actions.address (Add txt))
        ] []


view model =
    let
        entries = List.map viewEntry model.entries
    in
        Html.div [] ([ input , Html.text model.input ] ++ entries)

viewEntry entry =
    Html.div [] [ Html.text entry.content ]

main =
    Signal.map view (Signal.foldp update model actions.signal)
