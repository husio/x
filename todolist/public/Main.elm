import Html
import Html.Events
import Html.Attributes
import Signal

import Entry



type Action
    = NoOp
    | Add String
    | Remove Entry.ID
    | SetInput String
    | Update Entry.ID String
    | Edit Entry.ID Bool
    | SetState Entry.ID Entry.State
    | SetVisibility Visibility

type Visibility
    = All
    | Active
    | Completed


type alias Model =
    { entries : List Entry.Entry
    , visibility : Visibility
    , input : String
    , nextID : Entry.ID
    }

actions : Signal.Mailbox Action
actions =
    Signal.mailbox NoOp


model : Model
model =
    { entries = []
    , visibility = All
    , input = ""
    , nextID = 1
    }

update : Action -> Model -> Model
update action model =
    case action of
        NoOp ->
            model
        Add content ->
            let
                id = model.nextID
            in
                { model
                | entries = (Entry.new id content) :: model.entries
                , nextID = id + 1
                , input = ""
                }
        Remove id ->
            { model | entries = List.filter (\e -> e.id /= id) model.entries }
        SetInput content ->
            { model | input = content }
        Update id content ->
            let
                updateContent : Entry.ID -> String -> Entry.Entry -> Entry.Entry
                updateContent id content entry =
                    if entry.id == id then { entry | content = content } else entry
            in
                { model | entries = List.map (updateContent id content) model.entries }
        Edit id inEdit ->
            let
                updateEdit : Entry.ID -> Entry.Entry -> Entry.Entry
                updateEdit id entry =
                    if entry.id == id then { entry | inEdit = inEdit } else entry
            in
                { model | entries = List.map (updateEdit id) model.entries }
        SetState id state ->
            let
                updateState : Entry.ID -> Entry.Entry -> Entry.Entry
                updateState id entry =
                    if entry.id == id then { entry | state = state } else entry
            in
                { model | entries = List.map (updateState id) model.entries }
        SetVisibility All ->
            { model | visibility = All }
        SetVisibility Completed ->
            { model | visibility = Completed }
        SetVisibility Active ->
            { model | visibility = Active }


input : String -> Html.Html
input content =
    let
        keyAction key =
            if key == 13 then Add content else NoOp
        onKeyPress =
            Html.Events.onKeyDown actions.address keyAction

        setInput content =
            Signal.message actions.address (SetInput content)
        onInput =
            Html.Events.on "input" Html.Events.targetValue setInput
    in
        Html.input
            [ Html.Attributes.placeholder "content"
            , Html.Attributes.value content
            , onKeyPress
            , onInput
            ] []


view : Model -> Html.Html
view model =
    let
        visibleOnly entry =
            case model.visibility of
                All -> True
                Active -> entry.state == Entry.Active
                Completed -> entry.state == Entry.Completed
        entries = List.map viewEntry (List.filter visibleOnly model.entries)
    in
        Html.div [] (input model.input :: entries ++ [ viewFilters model.visibility ])

viewEntry : Entry.Entry -> Html.Html
viewEntry entry =
    let
        context : Entry.Context
        context =
            { remove = Signal.forwardTo actions.address (always (Remove entry.id))
            , change = Signal.forwardTo actions.address (Update entry.id)
            , edit = Signal.forwardTo actions.address (Edit entry.id)
            , state = Signal.forwardTo actions.address (SetState entry.id)
            }
    in
        Entry.view context entry


viewFilters : Visibility -> Html.Html
viewFilters visibility =
    let
        onClick : Visibility -> Html.Attribute
        onClick v = Html.Events.onClick actions.address (SetVisibility v)
    in
        Html.div []
            [ Html.button [ onClick All ] [ Html.text "all" ]
            , Html.button [ onClick Active ] [ Html.text "active" ]
            , Html.button [ onClick Completed ] [ Html.text "completed" ]
            ]

main =
    Signal.map view (Signal.foldp update model actions.signal)
