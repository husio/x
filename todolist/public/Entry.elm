module Entry where

import Html
import Html.Events
import Html.Attributes


type alias ID = Int

type State = Active | Completed

type alias Entry =
    { id : ID
    , content : String
    , inEdit: Bool
    , state : State
    }

new : ID -> String -> Entry
new id content =
    { id = id
    , content = content
    , inEdit = False
    , state = Active
    }


type alias Context =
    { change : Signal.Address String
    , remove : Signal.Address ()
    , edit : Signal.Address Bool
    , state : Signal.Address State
    }


view : Context -> Entry -> Html.Html
view context entry =
    let
        onRemove = Html.Events.onClick context.remove ()
        onEdit = Html.Events.onDoubleClick context.edit True
        onChange = Html.Events.on "input" Html.Events.targetValue (Signal.message context.change)
        onBlur = Html.Events.onBlur context.edit False
        onStateClick =
            case entry.state of
                Active -> Html.Events.onClick context.state Completed
                Completed -> Html.Events.onClick context.state Active
        contentView =
            if entry.inEdit then
               Html.input [ onChange, onBlur, Html.Attributes.value entry.content ] []
           else
               Html.span [ onEdit ] [ Html.text entry.content ]

    in
        Html.ul []
            [ Html.li [ ]
                [ (checkbox onStateClick (entry.state == Completed))
                , contentView
                , Html.button [ onRemove  ] [ Html.text "x" ]
                ]
            ]


checkbox onClick checked =
    Html.input
    [ Html.Attributes.type' "checkbox"
    , Html.Attributes.checked checked
    , onClick
    ]
    []
